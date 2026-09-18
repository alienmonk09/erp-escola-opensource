// Package autorizacao é o middleware único de RBAC (RS-020..023).
package autorizacao

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auditoria"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ctxKey int

const (
	CtxUsuario ctxKey = iota
	CtxPerfis
)

var Publicas = map[string]struct{}{
	"lerSaude":     {},
	"lerProntidao": {},
	"criarSessao":  {},
}

// Autenticadas exigem sessão (pré-login basta): logout, segundo fator e
// conta própria (cadastro TOTP guiado após o primeiro login — RF-002 —
// e troca com senha temporária — RF-004/RF-005).
var Autenticadas = map[string]struct{}{
	"encerrarSessao":     {},
	"confirmarTotp":      {},
	"gerirTotpProprio":   {},
	"trocarSenhaPropria": {},
}

var Escopos = map[string][]string{
	"redefinirSenhaUsuario": {"admin:usuarios:gerenciar"},
	"reiniciarTotpUsuario":  {"admin:usuarios:gerenciar"},
}

type Middleware struct {
	Sessao    *scs.SessionManager
	Queries   *db.Queries
	Auditoria auditoria.Registro
	Escopos   map[string][]string
}

var errInativo = errors.New("usuario inativo")

func (m Middleware) Handler(next http.Handler) http.Handler {
	escopos := m.Escopos
	if escopos == nil {
		escopos = Escopos
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opID := operationID(r.Method, r.URL.Path)
		if _, ok := Publicas[opID]; ok {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		if !m.Sessao.Exists(ctx, "usuario_id") {
			httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
			return
		}
		if _, ok := Autenticadas[opID]; ok {
			if auth.UsuarioID(ctx, m.Sessao) == "" {
				httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
				return
			}
			dono, err := m.donoDaSessao(ctx)
			if err != nil || revogada(ctx, m.Sessao, dono) {
				httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
				return
			}
			if opID == "trocarSenhaPropria" || opID == "encerrarSessao" {
				next.ServeHTTP(w, r)
				return
			}
			// Demais rotas de pré-login exigem troca em dia: com senha
			// temporária pendente, só a troca, o TOTP próprio e o logout
			// andam (RF-002/RF-004/RF-005).
			if opID != "gerirTotpProprio" && dono.TrocaObrigatoria {
				m.Auditoria.AcessoNegado(ctx, dono.ID, r)
				httpx.Escrever(w, http.StatusForbidden, httpapi.ErroErroCodigoSemPermissao, httpx.MsgTroca)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if !auth.Autenticado(ctx, m.Sessao) {
			httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
			return
		}
		uid := auth.UsuarioID(ctx, m.Sessao)
		dono, err := m.donoDaSessao(ctx)
		if err != nil || revogada(ctx, m.Sessao, dono) {
			httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
			return
		}
		// Troca obrigatória pendente: só troca, TOTP próprio e logout andam.
		if dono.TrocaObrigatoria && opID != "trocarSenhaPropria" && opID != "gerirTotpProprio" && opID != "encerrarSessao" {
			m.Auditoria.AcessoNegado(ctx, uuidParaPG(uid), r)
			httpx.Escrever(w, http.StatusForbidden, httpapi.ErroErroCodigoSemPermissao, httpx.MsgTroca)
			return
		}
		perfis, err := m.Queries.ListarCodigosPerfilDoUsuario(ctx, uuidParaPG(uid))
		if err != nil {
			httpx.Escrever(w, http.StatusInternalServerError, httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)
			return
		}
		// 2FA obrigatório pendente numa sessão efetivada (ex.: TOTP
		// reiniciado pelo admin sem derrubar esta sessão): só o cadastro
		// próprio, a troca e o logout andam (RF-002, RS-009).
		if opID != "gerirTotpProprio" && opID != "trocarSenhaPropria" && opID != "encerrarSessao" && totpObrigatorioPendente(dono, perfis) {
			m.Auditoria.AcessoNegado(ctx, uuidParaPG(uid), r)
			httpx.Escrever(w, http.StatusForbidden, httpapi.ErroErroCodigoSemPermissao, httpx.MsgTotp)
			return
		}
		exigidos := escopos[opID]
		if len(exigidos) > 0 && !temScope(perfis, exigidos) {
			m.Auditoria.AcessoNegado(ctx, uuidParaPG(uid), r)
			httpx.Escrever(w, http.StatusForbidden, httpapi.ErroErroCodigoSemPermissao, httpx.MsgSemPerm)
			return
		}
		if _, conhecida := Autenticadas[opID]; !conhecida && len(exigidos) == 0 {
			m.Auditoria.AcessoNegado(ctx, uuidParaPG(uid), r)
			httpx.Escrever(w, http.StatusForbidden, httpapi.ErroErroCodigoSemPermissao, httpx.MsgSemPerm)
			return
		}
		ctx = context.WithValue(ctx, CtxUsuario, uid)
		ctx = context.WithValue(ctx, CtxPerfis, perfis)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func operationID(method, pattern string) string {
	switch method + " " + pattern {
	case "GET /api/saude":
		return "lerSaude"
	case "GET /api/saude/pronto":
		return "lerProntidao"
	case "POST /api/sessao":
		return "criarSessao"
	case "DELETE /api/sessao":
		return "encerrarSessao"
	case "POST /api/sessao/totp":
		return "confirmarTotp"
	case "POST /api/conta/totp":
		return "gerirTotpProprio"
	case "POST /api/conta/senha":
		return "trocarSenhaPropria"
	default:
		// Rotas com {id}: /api/usuarios/<uuid>/senha-redefinicao e
		// /api/usuarios/<uuid>/totp-reinicio (o chi entrega o path real).
		if method == http.MethodPost && strings.HasPrefix(pattern, "/api/usuarios/") {
			switch {
			case strings.HasSuffix(pattern, "/senha-redefinicao"):
				return "redefinirSenhaUsuario"
			case strings.HasSuffix(pattern, "/totp-reinicio"):
				return "reiniciarTotpUsuario"
			}
		}
		return method + " " + pattern
	}
}

// donoDaSessao carrega o usuário dono da sessão (erro se sem sessão,
// inexistente ou inativo).
func (m Middleware) donoDaSessao(ctx context.Context) (db.Usuario, error) {
	u, err := m.Queries.GetUsuarioPorID(ctx, uuidParaPG(auth.UsuarioID(ctx, m.Sessao)))
	if err != nil {
		return db.Usuario{}, err
	}
	if !u.Ativo {
		return db.Usuario{}, errInativo
	}
	return u, nil
}

// revogada diz se a sessão morreu por troca/redefinição/reinício (RS-006).
func revogada(ctx context.Context, sess *scs.SessionManager, dono db.Usuario) bool {
	if !dono.SessoesInvalidasAntesDe.Valid {
		return false
	}
	return auth.Revogada(ctx, sess, dono.SessoesInvalidasAntesDe.Time.UnixNano(), true)
}

// totpObrigatorioPendente diz se perfil com 2FA obrigatório segue sem TOTP
// ativo (RF-002, RS-009): a sessão fica restrita ao cadastro guiado.
func totpObrigatorioPendente(dono db.Usuario, perfis []string) bool {
	return !dono.TotpAtivo && auth.ObrigatorioPara(perfis)
}

func temScope(perfis, exigidos []string) bool {
	set := map[string]struct{}{}
	for _, p := range perfis {
		for _, s := range ScopesDoPerfil(p) {
			set[s] = struct{}{}
		}
	}
	for _, e := range exigidos {
		if _, ok := set[e]; ok {
			return true
		}
	}
	return false
}

func ScopesDoPerfil(codigo string) []string {
	switch codigo {
	case "ADM":
		return []string{"admin:usuarios:gerenciar", "admin:auditoria:ler"}
	case "DIR":
		return []string{"admin:auditoria:ler", "gerencial:paineis:ler"}
	default:
		return nil
	}
}

func uuidParaPG(s string) pgtype.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func TemScope(perfis, exigidos []string) bool {
	return temScope(perfis, exigidos)
}
