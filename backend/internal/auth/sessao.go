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
	// Nanossegundos (não segundos): kills e logins no mesmo segundo precisam
	// de ordem total para RS-006 (ver Revogada).
	m.Put(ctx, chaveCriadaEm, time.Now().UTC().UnixNano())
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

// CriadaEm devolve o UnixNano da criação da sessão (gravado no login).
// Zero se ausente: tratada como válida pelo carimbo (ver Revogada).
func CriadaEm(ctx context.Context, m *scs.SessionManager) int64 {
	return m.GetInt64(ctx, chaveCriadaEm)
}

// Revogada diz se a sessão é mais antiga que o carimbo de invalidação do
// usuário (RS-006: troca/redefinição/reinício encerram sessões).
// invalidadasAntesDeNano é UnixNano do carimbo; valido=false (carimbo
// ausente) = nada invalidado. CriadaEm zero = válida.
func Revogada(ctx context.Context, m *scs.SessionManager, invalidadasAntesDeNano int64, valido bool) bool {
	if !valido {
		return false
	}
	criada := CriadaEm(ctx, m)
	if criada == 0 {
		return false
	}
	return criada < invalidadasAntesDeNano
}

// Efetivar promove a sessão pré-login a efetivada (segundo fator concluído):
// renova o ID (RS-005) e limpa o marcador de pré-login.
func Efetivar(ctx context.Context, m *scs.SessionManager) error {
	if err := m.RenewToken(ctx); err != nil {
		return err
	}
	m.Put(ctx, chavePreLogin, false)
	return nil
}
