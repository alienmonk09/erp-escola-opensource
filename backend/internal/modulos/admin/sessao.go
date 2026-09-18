// Package admin implementa as rotas de sessão da F-04 (TOTP em F-05).
package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/alexedwards/scs/v2"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auditoria"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type ctxReq struct{}

func ComRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxReq{}, r)))
	})
}

func requestDe(ctx context.Context) *http.Request {
	r, _ := ctx.Value(ctxReq{}).(*http.Request)
	if r == nil {
		return &http.Request{}
	}
	return r
}

type API struct {
	Pool      *pgxpool.Pool
	Queries   *db.Queries
	Sessao    *scs.SessionManager
	Auditoria auditoria.Registro
}

func (a API) LerSaude(ctx context.Context, _ httpapi.LerSaudeRequestObject) (httpapi.LerSaudeResponseObject, error) {
	return httpapi.LerSaude200JSONResponse{Status: httpapi.SaudeStatusOk}, nil
}

func (a API) LerProntidao(ctx context.Context, _ httpapi.LerProntidaoRequestObject) (httpapi.LerProntidaoResponseObject, error) {
	if err := a.Pool.Ping(ctx); err != nil {
		slog.Error("prontidao banco indisponivel")
		return httpapi.LerProntidao500JSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)), nil
	}
	return httpapi.LerProntidao200JSONResponse{
		Status: httpapi.ProntidaoStatusOk,
		Banco:  httpapi.ProntidaoBancoOk,
	}, nil
}

func (a API) CriarSessao(ctx context.Context, req httpapi.CriarSessaoRequestObject) (httpapi.CriarSessaoResponseObject, error) {
	r := requestDe(ctx)
	naoAuth := httpapi.CriarSessao401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgCredenciais)),
	}
	if req.Body == nil {
		return httpapi.CriarSessao400JSONResponse{
			PayloadInvalidoJSONResponse: httpapi.PayloadInvalidoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoPayloadInvalido, httpx.MsgPayload)),
		}, nil
	}
	email := strings.ToLower(strings.TrimSpace(string(req.Body.Email)))
	senha := req.Body.Senha

	bloqueada, err := auth.ContaBloqueada(ctx, a.Queries, email)
	if err != nil {
		slog.Error("bloqueio consulta falhou")
		return internoCriar(), nil
	}
	if bloqueada {
		a.Auditoria.LoginFalha(ctx, r)
		return naoAuth, nil
	}

	u, err := a.Queries.GetUsuarioPorEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("login consulta falhou")
			return internoCriar(), nil
		}
		_, _ = argon2id.CreateHash("x", argon2id.DefaultParams)
		_ = auth.RegistrarFalha(ctx, a.Queries, email)
		a.Auditoria.LoginFalha(ctx, r)
		return naoAuth, nil
	}
	ok, err := argon2id.ComparePasswordAndHash(senha, u.HashSenha)
	if err != nil || !ok || !u.Ativo {
		_ = auth.RegistrarFalha(ctx, a.Queries, email)
		a.Auditoria.LoginFalha(ctx, r)
		return naoAuth, nil
	}

	perfis, err := a.Queries.ListarCodigosPerfilDoUsuario(ctx, u.ID)
	if err != nil {
		slog.Error("perfis consulta falhou")
		return internoCriar(), nil
	}
	pede := pedeTOTP(u, perfis)
	uid := uuid.UUID(u.ID.Bytes).String()
	if err := auth.GravarLogin(ctx, a.Sessao, uid, pede); err != nil {
		slog.Error("sessao renovar falhou")
		return internoCriar(), nil
	}
	if token := a.Sessao.Token(ctx); token != "" {
		_ = a.Queries.VincularSessaoUsuario(ctx, db.VincularSessaoUsuarioParams{
			Token:     token,
			UsuarioID: u.ID,
		})
	}
	_ = auth.LimparFalhas(ctx, a.Queries, email)
	a.Auditoria.LoginOK(ctx, u.ID, r)

	out := httpapi.CriarSessao200JSONResponse{PedeTotp: pede}
	if !pede {
		id := openapi_types.UUID(u.ID.Bytes)
		out.UsuarioId = &id
	}
	return out, nil
}

func internoCriar() httpapi.CriarSessao500JSONResponse {
	return httpapi.CriarSessao500JSONResponse{
		ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
	}
}

func (a API) EncerrarSessao(ctx context.Context, _ httpapi.EncerrarSessaoRequestObject) (httpapi.EncerrarSessaoResponseObject, error) {
	if err := a.Sessao.Destroy(ctx); err != nil {
		slog.Error("logout falhou")
		return httpapi.EncerrarSessao500JSONResponse{
			ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
		}, nil
	}
	return httpapi.EncerrarSessao204Response{}, nil
}

func (a API) ConfirmarTotp(ctx context.Context, _ httpapi.ConfirmarTotpRequestObject) (httpapi.ConfirmarTotpResponseObject, error) {
	return httpapi.ConfirmarTotp401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgCredenciais)),
	}, nil
}

func pedeTOTP(u db.Usuario, perfis []string) bool {
	if u.TotpAtivo {
		return true
	}
	for _, p := range perfis {
		switch p {
		case "FIN", "RH", "DIR", "ADM":
			return true
		}
	}
	return false
}

// EncerrarTodas invalida sessões persistidas do usuário (RS-006).
func EncerrarTodas(ctx context.Context, q *db.Queries, usuarioID pgtype.UUID) error {
	return q.EncerrarSessoesDoUsuario(ctx, usuarioID)
}
