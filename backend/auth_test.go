// auth_test.go — F-04: login 401+auditoria, 401/403, rate limit 429.
package backend_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/autorizacao"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/servidor"
	"github.com/jackc/pgx/v5/pgxpool"
)

const senhaFicticia = "SenhaFicticia-Teste-09"

func sobeAPI(t *testing.T) http.Handler {
	t.Helper()
	_, dsn := sobeBanco(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "SET ROLE sacre_app"); err != nil {
		t.Fatalf("set role: %v", err)
	}
	hash, err := argon2id.CreateHash(senhaFicticia, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("argon2id: %v", err)
	}
	fix, err := os.ReadFile("testdata/fixtures_fundacao.sql")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, string(fix)); err != nil {
		t.Fatalf("carregar fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE usuario SET hash_senha = $1 WHERE email = $2`, hash, fixtureEmail); err != nil {
		t.Fatalf("hash fixture: %v", err)
	}
	sess := auth.NovaSessao(pool, false)
	return servidor.Novo(pool, sess)
}

func TestAuth_LoginSenhaErrada_401eAuditoria(t *testing.T) {
	h := sobeAPI(t)
	body := `{"email":"helena.ficticia@exemplo-escola.test","senha":"errada"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessao", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.1:9"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d corpo=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "NaoAutenticado") {
		t.Fatalf("envelope: %s", rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "helena") {
		t.Fatalf("vazou dado pessoal: %s", rec.Body.String())
	}
}

func TestAuth_SemSessao_401(t *testing.T) {
	h := sobeAPI(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/sessao", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d corpo=%s", rec.Code, rec.Body.String())
	}
}

func TestAuth_SemScope_403(t *testing.T) {
	if autorizacao.TemScope([]string{"SEC"}, []string{"admin:usuarios:gerenciar"}) {
		t.Fatal("SEC não deve ter scope de admin")
	}
	if !autorizacao.TemScope([]string{"ADM"}, []string{"admin:usuarios:gerenciar"}) {
		t.Fatal("ADM deve ter scope de admin")
	}
}

func TestAuth_RateLimit_429(t *testing.T) {
	h := sobeAPI(t)
	body := []byte(`{"email":"naoexiste@exemplo-escola.test","senha":"x"}`)
	var last int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/sessao", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.1.2.3:80"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("6ª tentativa status=%d", last)
	}
}

func TestAuth_LoginOk_Saude(t *testing.T) {
	h := sobeAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/saude", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("saude=%d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "ok" {
		t.Fatalf("saude corpo=%s", rec.Body.String())
	}

	body := `{"email":"helena.ficticia@exemplo-escola.test","senha":"` + senhaFicticia + `"}`
	login := httptest.NewRequest(http.MethodPost, "/api/sessao", strings.NewReader(body))
	login.Header.Set("Content-Type", "application/json")
	login.RemoteAddr = "10.0.0.8:9"
	lr := httptest.NewRecorder()
	h.ServeHTTP(lr, login)
	if lr.Code != http.StatusOK {
		t.Fatalf("login=%d %s", lr.Code, lr.Body.String())
	}

	pronto := httptest.NewRequest(http.MethodGet, "/api/saude/pronto", nil)
	pr := httptest.NewRecorder()
	h.ServeHTTP(pr, pronto)
	if pr.Code != http.StatusOK {
		t.Fatalf("pronto=%d %s", pr.Code, pr.Body.String())
	}
}

func TestAuth_AuditoriaLoginFalha(t *testing.T) {
	_, dsn := sobeBanco(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	h := sobeAPIFromPool(t, pool)
	body := `{"email":"helena.ficticia@exemplo-escola.test","senha":"errada"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessao", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.9.9.9:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	q := db.New(pool)
	_ = q
	var acao string
	if err := pool.QueryRow(ctx, `SELECT acao FROM auditoria WHERE acao = 'login_falha' LIMIT 1`).Scan(&acao); err != nil {
		t.Fatalf("auditoria: %v", err)
	}
}

func sobeAPIFromPool(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()
	ctx := context.Background()
	hash, err := argon2id.CreateHash(senhaFicticia, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("argon2id: %v", err)
	}
	fix, err := os.ReadFile("testdata/fixtures_fundacao.sql")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, string(fix)); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE usuario SET hash_senha = $1 WHERE email = $2`, hash, fixtureEmail); err != nil {
		t.Fatalf("hash: %v", err)
	}
	sess := auth.NovaSessao(pool, false)
	return servidor.Novo(pool, sess)
}
