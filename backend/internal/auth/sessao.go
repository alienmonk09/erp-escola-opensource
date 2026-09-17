// Package auth configura scs + pgxstore (RS-003, RS-005).
package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	chaveUsuarioID = "usuario_id"
	chaveCriadaEm  = "criada_em"
	chavePreLogin  = "pre_login"
)

func NovaSessao(pool *pgxpool.Pool, cookieSecure bool) *scs.SessionManager {
	m := scs.New()
	m.Store = pgxstore.New(pool)
	m.Lifetime = config.Absoluto
	m.IdleTimeout = config.Inatividade
	m.Cookie.Name = config.CookieNome
	m.Cookie.HttpOnly = true
	m.Cookie.Secure = cookieSecure
	m.Cookie.SameSite = http.SameSiteStrictMode
	m.Cookie.Path = "/"
	m.Cookie.Persist = true
	return m
}

func GravarLogin(ctx context.Context, m *scs.SessionManager, usuarioID string, preLogin bool) error {
	if err := m.RenewToken(ctx); err != nil {
		return err
	}
	m.Put(ctx, chaveUsuarioID, usuarioID)
	m.Put(ctx, chaveCriadaEm, time.Now().UTC().Unix())
	m.Put(ctx, chavePreLogin, preLogin)
	return nil
}

func UsuarioID(ctx context.Context, m *scs.SessionManager) string {
	return m.GetString(ctx, chaveUsuarioID)
}

func PreLogin(ctx context.Context, m *scs.SessionManager) bool {
	return m.GetBool(ctx, chavePreLogin)
}

func Autenticado(ctx context.Context, m *scs.SessionManager) bool {
	id := UsuarioID(ctx, m)
	return id != "" && !PreLogin(ctx, m)
}
