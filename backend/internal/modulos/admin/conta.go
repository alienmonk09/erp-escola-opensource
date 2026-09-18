// Package admin — conta própria e gestão de usuários da F-05.
//
// Requisitos: RF-003, RF-005, RF-007; RS-006..009; RS-010 (auditoria).
// TOTP via backend/internal/auth (pquerna/otp + secretbox); senhas só com
// alexedwards/argon2id (RS-001). Segredos e códigos nunca em log (RS-066);
// códigos TOTP nunca persistidos (só validados em memória).
package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexedwards/argon2id"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// usuarioDaSessao carrega o dono da sessão (401 se sem sessão, inexistente ou
// inativo — sem revelar o motivo, RS-002) com seus perfis.
func (a API) usuarioDaSessao(ctx context.Context) (db.Usuario, []string, error) {
	uid := auth.UsuarioID(ctx, a.Sessao)
	if uid == "" {
		return db.Usuario{}, nil, errNaoAuth
	}
	id, err := uuid.Parse(uid)
	if err != nil {
		return db.Usuario{}, nil, errNaoAuth
	}
	u, err := a.Queries.GetUsuarioPorID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("conta consulta falhou")
		}
		return db.Usuario{}, nil, errNaoAuth
	}
	if !u.Ativo {
		return db.Usuario{}, nil, errNaoAuth
	}
	perfis, err := a.Queries.ListarCodigosPerfilDoUsuario(ctx, u.ID)
	if err != nil {
		slog.Error("conta perfis falhou")
		return db.Usuario{}, nil, errInterno
	}
	return u, perfis, nil
}

var errNaoAuth = errors.New("nao autenticado")
var errInterno = errors.New("erro interno")

// GerirTotpProprio implementa POST /api/conta/totp (RF-003, RS-009).
func (a API) GerirTotpProprio(ctx context.Context, req httpapi.GerirTotpProprioRequestObject) (httpapi.GerirTotpProprioResponseObject, error) {
	r := requestDe(ctx)
	if req.Body == nil {
		return payloadGerir(), nil
	}
	u, perfis, err := a.usuarioDaSessao(ctx)
	if err != nil {
		if errors.Is(err, errInterno) {
			return internoGerir(), nil
		}
		return naoAuthGerir(), nil
	}

	switch req.Body.Acao {
	case httpapi.Iniciar:
		return a.totpIniciar(ctx, r, u)
	case httpapi.Confirmar:
		return a.totpConfirmar(ctx, r, u, codigoDe(req.Body))
	case httpapi.Desativar:
		return a.totpDesativar(ctx, r, u, perfis, codigoDe(req.Body))
	default:
		return payloadGerir(), nil
	}
}

func codigoDe(body *httpapi.GerirTotpProprioJSONRequestBody) string {
	if body == nil || body.Codigo == nil {
		return ""
	}
	return *body.Codigo
}

func (a API) totpIniciar(ctx context.Context, r *http.Request, u db.Usuario) (httpapi.GerirTotpProprioResponseObject, error) {
	cifrado, claro, url, err := auth.GerarSegredo(u.Email)
	if err != nil {
		slog.Error("totp gerar falhou")
		return internoGerir(), nil
	}
	if err := a.Queries.DefinirTotpSecreto(ctx, db.DefinirTotpSecretoParams{
		ID:                 u.ID,
		TotpSecretoCifrado: cifrado,
	}); err != nil {
		slog.Error("totp guardar falhou")
		return internoGerir(), nil
	}
	out := httpapi.ContaTotpResposta{Segredo: &claro, OtpauthUrl: &url}
	return httpapi.GerirTotpProprio200JSONResponse(out), nil
}

func (a API) totpConfirmar(ctx context.Context, r *http.Request, u db.Usuario, codigo string) (httpapi.GerirTotpProprioResponseObject, error) {
	if !auth.ValidarFormato(codigo) {
		return payloadGerir(), nil
	}
	if u.SessoesInvalidasAntesDe.Valid &&
		auth.Revogada(ctx, a.Sessao, u.SessoesInvalidasAntesDe.Time.UnixNano(), true) {
		// Sessão encerrada por troca/redefinição/reinício (RS-006).
		a.Auditoria.TotpFalha(ctx, u.ID, r)
		return naoAuthGerir(), nil
	}
	if len(u.TotpSecretoCifrado) == 0 {
		a.Auditoria.TotpFalha(ctx, u.ID, r)
		return naoAuthGerir(), nil
	}
	claro, err := auth.DecifrarSegredo(u.TotpSecretoCifrado)
	if err != nil {
		slog.Error("totp decifrar falhou")
		return internoGerir(), nil
	}
	if !auth.ValidarCodigo(claro, codigo) {
		a.Auditoria.TotpFalha(ctx, u.ID, r)
		return naoAuthGerir(), nil
	}
	if err := a.Queries.AtivarTotp(ctx, u.ID); err != nil {
		slog.Error("totp ativar falhou")
		return internoGerir(), nil
	}
	if err := auth.Efetivar(ctx, a.Sessao); err != nil {
		slog.Error("totp efetivar falhou")
		return internoGerir(), nil
	}
	a.Auditoria.TotpCadastrado(ctx, u.ID, r)
	a.Auditoria.TotpOK(ctx, u.ID, r)
	falso := false
	id := openapi_types.UUID(u.ID.Bytes)
	return httpapi.GerirTotpProprio200JSONResponse(httpapi.ContaTotpResposta{
		PedeTotp:  &falso,
		UsuarioId: &id,
	}), nil
}

func (a API) totpDesativar(ctx context.Context, r *http.Request, u db.Usuario, perfis []string, codigo string) (httpapi.GerirTotpProprioResponseObject, error) {
	if auth.ObrigatorioPara(perfis) {
		a.Auditoria.AcessoNegado(ctx, u.ID, r)
		return httpapi.GerirTotpProprio403JSONResponse{
			SemPermissaoJSONResponse: httpapi.SemPermissaoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoSemPermissao, httpx.MsgSemPerm)),
		}, nil
	}
	if !auth.ValidarFormato(codigo) {
		return payloadGerir(), nil
	}
	if !u.TotpAtivo || len(u.TotpSecretoCifrado) == 0 {
		return payloadGerir(), nil
	}
	claro, err := auth.DecifrarSegredo(u.TotpSecretoCifrado)
	if err != nil {
		slog.Error("totp decifrar falhou")
		return internoGerir(), nil
	}
	if !auth.ValidarCodigo(claro, codigo) {
		a.Auditoria.TotpFalha(ctx, u.ID, r)
		return naoAuthGerir(), nil
	}
	if err := a.Queries.ReiniciarTotp(ctx, u.ID); err != nil {
		slog.Error("totp desativar falhou")
		return internoGerir(), nil
	}
	a.Auditoria.TotpDesativado(ctx, u.ID, r)
	falso := false
	return httpapi.GerirTotpProprio200JSONResponse(httpapi.ContaTotpResposta{PedeTotp: &falso}), nil
}

func payloadGerir() httpapi.GerirTotpProprioResponseObject {
	return httpapi.GerirTotpProprio400JSONResponse{
		PayloadInvalidoJSONResponse: httpapi.PayloadInvalidoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoPayloadInvalido, httpx.MsgPayload)),
	}
}

func naoAuthGerir() httpapi.GerirTotpProprioResponseObject {
	return httpapi.GerirTotpProprio401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgCredenciais)),
	}
}

func internoGerir() httpapi.GerirTotpProprioResponseObject {
	return httpapi.GerirTotpProprio500JSONResponse{
		ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
	}
}

// TrocarSenhaPropria implementa POST /api/conta/senha (RS-006).
func (a API) TrocarSenhaPropria(ctx context.Context, req httpapi.TrocarSenhaPropriaRequestObject) (httpapi.TrocarSenhaPropriaResponseObject, error) {
	r := requestDe(ctx)
	if req.Body == nil || req.Body.SenhaAtual == "" || req.Body.SenhaNova == "" {
		return payloadTrocar(), nil
	}
	u, _, err := a.usuarioDaSessao(ctx)
	if err != nil {
		if errors.Is(err, errInterno) {
			return internoTrocar(), nil
		}
		return naoAuthTrocar(), nil
	}
	ok, err := argon2id.ComparePasswordAndHash(req.Body.SenhaAtual, u.HashSenha)
	if err != nil || !ok {
		return naoAuthTrocar(), nil
	}
	if req.Body.SenhaNova == req.Body.SenhaAtual {
		return payloadTrocar(), nil
	}
	hash, err := argon2id.CreateHash(req.Body.SenhaNova, argon2id.DefaultParams)
	if err != nil {
		slog.Error("senha hash falhou")
		return internoTrocar(), nil
	}
	if err := a.Queries.TrocarSenhaPropria(ctx, db.TrocarSenhaPropriaParams{
		ID:        u.ID,
		HashSenha: hash,
	}); err != nil {
		slog.Error("senha trocar falhou")
		return internoTrocar(), nil
	}
	// RS-006: troca encerra TODAS as sessões (inclui a atual).
	if err := EncerrarTodas(ctx, a.Queries, u.ID); err != nil {
		slog.Error("senha encerrar sessoes falhou")
		return internoTrocar(), nil
	}
	_ = a.Sessao.Destroy(ctx)
	a.Auditoria.SenhaTrocada(ctx, u.ID, r)
	return httpapi.TrocarSenhaPropria204Response{}, nil
}

func payloadTrocar() httpapi.TrocarSenhaPropriaResponseObject {
	return httpapi.TrocarSenhaPropria400JSONResponse{
		PayloadInvalidoJSONResponse: httpapi.PayloadInvalidoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoPayloadInvalido, httpx.MsgPayload)),
	}
}

func naoAuthTrocar() httpapi.TrocarSenhaPropriaResponseObject {
	return httpapi.TrocarSenhaPropria401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgCredenciais)),
	}
}

func internoTrocar() httpapi.TrocarSenhaPropriaResponseObject {
	return httpapi.TrocarSenhaPropria500JSONResponse{
		ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
	}
}

// RedefinirSenhaUsuario implementa POST /api/usuarios/{id}/senha-redefinicao
// (RF-005, RS-007). Escopo fiscalizado no middleware; aqui só a ação.
func (a API) RedefinirSenhaUsuario(ctx context.Context, req httpapi.RedefinirSenhaUsuarioRequestObject) (httpapi.RedefinirSenhaUsuarioResponseObject, error) {
	r := requestDe(ctx)
	admin, _, err := a.usuarioDaSessao(ctx)
	if err != nil {
		if errors.Is(err, errInterno) {
			return internoRedefinir(), nil
		}
		return naoAuthRedefinir(), nil
	}
	alvo, err := a.Queries.GetUsuarioPorID(ctx, pgtype.UUID{Bytes: req.Id, Valid: true})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("redefinicao consulta falhou")
			return internoRedefinir(), nil
		}
		return httpapi.RedefinirSenhaUsuario404JSONResponse{
			NaoEncontradoJSONResponse: httpapi.NaoEncontradoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoEncontrado, httpx.MsgSemPerm)),
		}, nil
	}
	temporaria, err := auth.GerarSenhaTemporaria()
	if err != nil {
		slog.Error("redefinicao gerar falhou")
		return internoRedefinir(), nil
	}
	hash, err := argon2id.CreateHash(temporaria, argon2id.DefaultParams)
	if err != nil {
		slog.Error("redefinicao hash falhou")
		return internoRedefinir(), nil
	}
	if err := a.Queries.RedefinirSenhaAdmin(ctx, db.RedefinirSenhaAdminParams{
		ID:        alvo.ID,
		HashSenha: hash,
	}); err != nil {
		slog.Error("redefinicao gravar falhou")
		return internoRedefinir(), nil
	}
	if err := EncerrarTodas(ctx, a.Queries, alvo.ID); err != nil {
		slog.Error("redefinicao encerrar sessoes falhou")
		return internoRedefinir(), nil
	}
	if alvo.ID == admin.ID {
		_ = a.Sessao.Destroy(ctx)
	}
	a.Auditoria.SenhaRedefinida(ctx, admin.ID, uuid.UUID(alvo.ID.Bytes).String(), r)
	return httpapi.RedefinirSenhaUsuario200JSONResponse{SenhaTemporaria: temporaria}, nil
}

func naoAuthRedefinir() httpapi.RedefinirSenhaUsuarioResponseObject {
	return httpapi.RedefinirSenhaUsuario401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)),
	}
}

func internoRedefinir() httpapi.RedefinirSenhaUsuarioResponseObject {
	return httpapi.RedefinirSenhaUsuario500JSONResponse{
		ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
	}
}

// ReiniciarTotpUsuario implementa POST /api/usuarios/{id}/totp-reinicio
// (RF-007, RS-008). O 2FA obrigatório não é desligado, só reiniciado:
// o usuário refaz o cadastro guiado no próximo login (RF-002).
func (a API) ReiniciarTotpUsuario(ctx context.Context, req httpapi.ReiniciarTotpUsuarioRequestObject) (httpapi.ReiniciarTotpUsuarioResponseObject, error) {
	r := requestDe(ctx)
	admin, _, err := a.usuarioDaSessao(ctx)
	if err != nil {
		if errors.Is(err, errInterno) {
			return internoReiniciar(), nil
		}
		return naoAuthReiniciar(), nil
	}
	alvo, err := a.Queries.GetUsuarioPorID(ctx, pgtype.UUID{Bytes: req.Id, Valid: true})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("totp reinicio consulta falhou")
			return internoReiniciar(), nil
		}
		return httpapi.ReiniciarTotpUsuario404JSONResponse{
			NaoEncontradoJSONResponse: httpapi.NaoEncontradoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoEncontrado, httpx.MsgSemPerm)),
		}, nil
	}
	if err := a.Queries.ReiniciarTotp(ctx, alvo.ID); err != nil {
		slog.Error("totp reinicio gravar falhou")
		return internoReiniciar(), nil
	}
	if err := EncerrarTodas(ctx, a.Queries, alvo.ID); err != nil {
		slog.Error("totp reinicio encerrar sessoes falhou")
		return internoReiniciar(), nil
	}
	if alvo.ID == admin.ID {
		_ = a.Sessao.Destroy(ctx)
	}
	a.Auditoria.TotpReiniciado(ctx, admin.ID, uuid.UUID(alvo.ID.Bytes).String(), r)
	return httpapi.ReiniciarTotpUsuario204Response{}, nil
}

func naoAuthReiniciar() httpapi.ReiniciarTotpUsuarioResponseObject {
	return httpapi.ReiniciarTotpUsuario401JSONResponse{
		NaoAutenticadoJSONResponse: httpapi.NaoAutenticadoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)),
	}
}

func internoReiniciar() httpapi.ReiniciarTotpUsuarioResponseObject {
	return httpapi.ReiniciarTotpUsuario500JSONResponse{
		ErroInternoJSONResponse: httpapi.ErroInternoJSONResponse(httpx.Envelope(httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)),
	}
}
