// Package servidor monta o chi router da F-04.
package servidor

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auditoria"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/autorizacao"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpx"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/modulos/admin"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unrolled/secure"
)

func Novo(pool *pgxpool.Pool, sess *scs.SessionManager) http.Handler {
	q := db.New(pool)
	aud := auditoria.Registro{Queries: q}
	api := admin.API{Pool: pool, Queries: q, Sessao: sess, Auditoria: aud}

	r := chi.NewRouter()
	r.Use(cabecalhos())
	r.Use(csrfMutacoes)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Rate limit (RS-050/051): login e fluxos sensíveis da F-05
			// (TOTP, troca, redefinição e reinício) com contador no PG.
			if loginOuSensivel(req.Method, req.URL.Path) {
				auth.LimitadorLogin(q).Handler(next).ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Use(sess.LoadAndSave)
	r.Use(admin.ComRequest)
	r.Use(autorizacao.Middleware{Sessao: sess, Queries: q, Auditoria: aud}.Handler)

	strict := httpapi.NewStrictHandlerWithOptions(api, nil, httpapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			httpx.Escrever(w, http.StatusBadRequest, httpapi.ErroErroCodigoPayloadInvalido, httpx.MsgPayload)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			slog.Error("handler", "erro_tipo", "interno")
			httpx.Escrever(w, http.StatusInternalServerError, httpapi.ErroErroCodigoErroInterno, httpx.MsgInterno)
			_ = json.NewEncoder(w)
			_ = err
		},
	})
	return httpapi.HandlerWithOptions(strict, httpapi.ChiServerOptions{BaseRouter: r})
}

// loginOuSensivel diz se a rota entra no rate limit por IP com contador no
// PG (RS-050/051): login, validação TOTP, conta própria e gestão de usuários.
func loginOuSensivel(metodo, caminho string) bool {
	if metodo != http.MethodPost {
		return false
	}
	switch caminho {
	case "/api/sessao", "/api/sessao/totp", "/api/conta/totp", "/api/conta/senha":
		return true
	}
	if strings.HasPrefix(caminho, "/api/usuarios/") &&
		(strings.HasSuffix(caminho, "/senha-redefinicao") || strings.HasSuffix(caminho, "/totp-reinicio")) {
		return true
	}
	return false
}

func cabecalhos() func(http.Handler) http.Handler {
	s := secure.New(secure.Options{
		STSSeconds:            31536000,
		STSIncludeSubdomains:  true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'",
		PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
		IsDevelopment:         false,
	})
	return s.Handler
}

func csrfMutacoes(next http.Handler) http.Handler {
	prot := http.NewCrossOriginProtection()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		prot.Handler(next).ServeHTTP(w, r)
	})
}
