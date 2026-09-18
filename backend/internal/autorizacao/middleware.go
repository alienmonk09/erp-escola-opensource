// Package autorizacao é o middleware único de RBAC (RS-020..023).
package autorizacao

import (
	"context"
	"net/http"

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

var Autenticadas = map[string]struct{}{
	"encerrarSessao": {},
	"confirmarTotp":  {},
}

var Escopos = map[string][]string{}

type Middleware struct {
	Sessao    *scs.SessionManager
	Queries   *db.Queries
	Auditoria auditoria.Registro
	Escopos   map[string][]string
}

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
		if opID == "confirmarTotp" || opID == "encerrarSessao" {
			if auth.UsuarioID(ctx, m.Sessao) == "" {
				httpx.Escrever(w, http.StatusUnauthorized, httpapi.ErroErroCodigoNaoAutenticado, httpx.MsgNaoAuth)
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
		perfis, err := m.Queries.ListarCodigosPerfilDoUsuario(ctx, uuidParaPG(uid))
		if err != nil {
			httpx.Escrever(w, http.StatusInternalServerError, httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)
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
	default:
		return method + " " + pattern
	}
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
