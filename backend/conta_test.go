// conta_test.go — F-05: TOTP obrigatório/opcional, troca, redefinição e reinício.
//
// Requisitos cobertos: RF-003, RF-005, RF-007; RS-006..009; RS-010.
// Só dados fictícios (RE-06, domínio .test); chave TOTP efêmera por teste via
// crypto/rand + t.Setenv — nenhum segredo em código/fixture (RS-060/066).
package backend_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/servidor"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaveTotpTeste instala uma TOTP_ENCRYPTION_KEY aleatória (32 bytes em
// base64) válida só neste teste. Sem valor fixo no repo (RS-060/066).
func chaveTotpTeste(t *testing.T) {
	t.Helper()
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("aleatorio: %v", err)
	}
	t.Setenv("TOTP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(raw[:]))
}

// jar é um cliente HTTP mínimo com cookie jar para encadear login → ações.
type jar struct {
	h       http.Handler
	cookies map[string]string
}

func novoJar(h http.Handler) *jar {
	return &jar{h: h, cookies: map[string]string{}}
}

func (c *jar) faz(t *testing.T, metodo, caminho, corpo, ip string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if corpo == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(corpo)
	}
	req := httptest.NewRequest(metodo, caminho, reader)
	if corpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = ip + ":9"
	var jarra []string
	for n, v := range c.cookies {
		jarra = append(jarra, n+"="+v)
	}
	if len(jarra) > 0 {
		req.Header.Set("Cookie", strings.Join(jarra, "; "))
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	for _, sc := range rec.Result().Cookies() {
		c.cookies[sc.Name] = sc.Value
		if sc.MaxAge < 0 || (sc.Value == "" && sc.MaxAge == 0) {
			delete(c.cookies, sc.Name)
		}
	}
	return rec
}

func corpoJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("resposta não-JSON (%d): %s", rec.Code, rec.Body.String())
	}
	return out
}

// sobeConta cria banco + API e devolve pool e handler (papel app assumido).
func sobeConta(t *testing.T) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	chaveTotpTeste(t)
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
	return pool, servidor.Novo(pool, auth.NovaSessao(pool, false))
}

// criarUsuario insere usuário fictício com hash argon2id válido e perfis.
func criarUsuario(t *testing.T, pool *pgxpool.Pool, nome, email, senha string, perfis ...string) string {
	t.Helper()
	ctx := context.Background()
	hash, err := argon2id.CreateHash(senha, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("argon2id: %v", err)
	}
	id := uuid.NewString()
	if _, err := pool.Exec(ctx,
		`INSERT INTO usuario (id, nome, email, hash_senha) VALUES ($1, $2, $3, $4)`,
		id, nome, email, hash); err != nil {
		t.Fatalf("inserir usuario: %v", err)
	}
	for _, p := range perfis {
		if _, err := pool.Exec(ctx,
			`INSERT INTO usuario_perfil (usuario_id, perfil_id) VALUES ($1, (SELECT id FROM perfil WHERE codigo = $2))`,
			id, p); err != nil {
			t.Fatalf("perfil %s: %v", p, err)
		}
	}
	return id
}

// entregaAdmin devolve um jar com sessão ADM efetivada (TOTP cadastrado).
func entregaAdmin(t *testing.T, pool *pgxpool.Pool, h http.Handler) *jar {
	t.Helper()
	const senha = "SenhaFicticia-Admin-31"
	criarUsuario(t, pool, "Ada Admin Fictícia", "ada.admin.ficticia@exemplo-escola.test", senha, "ADM")
	a := novoJar(h)
	rec := a.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"ada.admin.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.20.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("login admin=%d %s", rec.Code, rec.Body.String())
	}
	rec = a.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.20.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("totp iniciar admin=%d %s", rec.Code, rec.Body.String())
	}
	segredo := corpoJSON(t, rec)["segredo"].(string)
	codigo, err := auth.CodigoAtual(segredo)
	if err != nil {
		t.Fatalf("codigo: %v", err)
	}
	rec = a.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"confirmar","codigo":"`+codigo+`"}`, "10.20.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("totp confirmar admin=%d %s", rec.Code, rec.Body.String())
	}
	return a
}

func auditoriaExiste(t *testing.T, pool *pgxpool.Pool, acao string) bool {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM auditoria WHERE acao = $1`, acao).Scan(&n); err != nil {
		t.Fatalf("auditoria: %v", err)
	}
	return n > 0
}

// Given FIN sem TOTP, When login só com senha, Then bloqueado até cadastrar.
func TestConta_FinSemTotp_BloqueadoAteCadastrar(t *testing.T) {
	pool, h := sobeConta(t)
	const senha = "SenhaFicticia-Fin-17"
	criarUsuario(t, pool, "Fiona Financeiro Fictícia", "fiona.fin.ficticia@exemplo-escola.test", senha, "FIN")

	c := novoJar(h)
	rec := c.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"fiona.fin.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.30.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("login=%d %s", rec.Code, rec.Body.String())
	}
	if corpoJSON(t, rec)["pedeTotp"] != true {
		t.Fatalf("FIN sem TOTP deveria pedir TOTP: %s", rec.Body.String())
	}

	// Só senha não efetiva: validar código sem cadastro → 401.
	rec = c.faz(t, http.MethodPost, "/api/sessao/totp", `{"codigo":"123456"}`, "10.30.0.1")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("totp sem cadastro deveria ser 401, deu %d", rec.Code)
	}

	// Cadastro guiado: iniciar devolve segredo + URL, sem vazar em auditoria.
	rec = c.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.30.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("iniciar=%d %s", rec.Code, rec.Body.String())
	}
	inicio := corpoJSON(t, rec)
	segredo, _ := inicio["segredo"].(string)
	url, _ := inicio["otpauthUrl"].(string)
	if segredo == "" || !strings.HasPrefix(url, "otpauth://") {
		t.Fatalf("iniciar sem segredo/url: %s", rec.Body.String())
	}

	codigo, err := auth.CodigoAtual(segredo)
	if err != nil {
		t.Fatalf("codigo: %v", err)
	}
	rec = c.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"confirmar","codigo":"`+codigo+`"}`, "10.30.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmar=%d %s", rec.Code, rec.Body.String())
	}
	if corpoJSON(t, rec)["pedeTotp"] != false {
		t.Fatalf("após cadastro pedeTotp deveria ser false: %s", rec.Body.String())
	}

	// Sessão efetivada: troca de senha própria funciona.
	rec = c.faz(t, http.MethodPost, "/api/conta/senha",
		`{"senhaAtual":"`+senha+`","senhaNova":"SenhaFicticia-Fin-18"}`, "10.30.0.1")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("troca após efetivar=%d %s", rec.Code, rec.Body.String())
	}
	if !auditoriaExiste(t, pool, "totp_cadastrado") || !auditoriaExiste(t, pool, "totp_ok") {
		t.Fatal("auditoria de TOTP ausente")
	}
}

// Given redefinição por admin, When usuário troca a senha,
// Then sessões antigas invalidadas + auditoria.
func TestConta_Redefinicao_Troca_InvalidaSessoes(t *testing.T) {
	pool, h := sobeConta(t)
	const senha = "SenhaFicticia-Sec-22"
	alvoID := criarUsuario(t, pool, "Sara Secretaria Fictícia", "sara.sec.ficticia@exemplo-escola.test", senha, "SEC")
	admin := entregaAdmin(t, pool, h)

	// Sessão antiga do usuário (antes da redefinição).
	u := novoJar(h)
	rec := u.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"sara.sec.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.40.0.2")
	if rec.Code != http.StatusOK {
		t.Fatalf("login usuario=%d %s", rec.Code, rec.Body.String())
	}

	// Admin redefine: devolve temporária de uso único.
	rec = admin.faz(t, http.MethodPost, "/api/usuarios/"+alvoID+"/senha-redefinicao", "", "10.40.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("redefinicao=%d %s", rec.Code, rec.Body.String())
	}
	temporaria, _ := corpoJSON(t, rec)["senhaTemporaria"].(string)
	if temporaria == "" {
		t.Fatalf("sem temporária: %s", rec.Body.String())
	}

	// Sessão antiga morreu na redefinição.
	rec = u.faz(t, http.MethodPost, "/api/conta/senha",
		`{"senhaAtual":"`+senha+`","senhaNova":"x"}`, "10.40.0.2")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sessão antiga deveria morrer, deu %d", rec.Code)
	}

	// Login com a temporária exige troca: segundo fator bloqueia com 403.
	u2 := novoJar(h)
	rec = u2.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"sara.sec.ficticia@exemplo-escola.test","senha":"`+temporaria+`"}`, "10.40.0.3")
	if rec.Code != http.StatusOK {
		t.Fatalf("login temporária=%d %s", rec.Code, rec.Body.String())
	}
	rec = u2.faz(t, http.MethodPost, "/api/sessao/totp", `{"codigo":"000000"}`, "10.40.0.3")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("com troca pendente, totp deveria ser 403, deu %d", rec.Code)
	}

	// Troca encerra a sessão da temporária (prova server-side: com o cookie
	// original ainda presente, a sessão não existe mais).
	const nova = "SenhaFicticia-Sec-23"
	preservado := map[string]string{}
	for n, v := range u2.cookies {
		preservado[n] = v
	}
	rec = u2.faz(t, http.MethodPost, "/api/conta/senha",
		`{"senhaAtual":"`+temporaria+`","senhaNova":"`+nova+`"}`, "10.40.0.3")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("troca=%d %s", rec.Code, rec.Body.String())
	}
	u2.cookies = preservado
	rec = u2.faz(t, http.MethodPost, "/api/conta/senha",
		`{"senhaAtual":"`+nova+`","senhaNova":"outra"}`, "10.40.0.3")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sessão da temporária deveria morrer, deu %d", rec.Code)
	}

	// Temporária não serve mais; nova senha funciona.
	u3 := novoJar(h)
	rec = u3.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"sara.sec.ficticia@exemplo-escola.test","senha":"`+temporaria+`"}`, "10.40.0.4")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("temporária reutilizada deveria ser 401, deu %d", rec.Code)
	}
	rec = u3.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"sara.sec.ficticia@exemplo-escola.test","senha":"`+nova+`"}`, "10.40.0.4")
	if rec.Code != http.StatusOK {
		t.Fatalf("login nova=%d %s", rec.Code, rec.Body.String())
	}
	if !auditoriaExiste(t, pool, "senha_redefinida") || !auditoriaExiste(t, pool, "senha_trocada") {
		t.Fatal("auditoria de senha ausente")
	}
}

// TOTP perdido: admin reinicia, sessões caem, próximo login pede cadastro.
func TestConta_TotpReinicio_PedeCadastroDeNovo(t *testing.T) {
	pool, h := sobeConta(t)
	const senha = "SenhaFicticia-Fin-41"
	alvoID := criarUsuario(t, pool, "Rita RH Fictícia", "rita.rh.ficticia@exemplo-escola.test", senha, "RH")
	admin := entregaAdmin(t, pool, h)

	// Usuária cadastra o TOTP e efetiva.
	u := novoJar(h)
	u.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"rita.rh.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.50.0.2")
	rec := u.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.50.0.2")
	segredo := corpoJSON(t, rec)["segredo"].(string)
	codigo, _ := auth.CodigoAtual(segredo)
	if rec := u.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"confirmar","codigo":"`+codigo+`"}`, "10.50.0.2"); rec.Code != http.StatusOK {
		t.Fatalf("confirmar=%d %s", rec.Code, rec.Body.String())
	}

	// Admin reinicia após revalidação (procedimento RS-008, fora da API).
	if rec := admin.faz(t, http.MethodPost, "/api/usuarios/"+alvoID+"/totp-reinicio", "", "10.50.0.1"); rec.Code != http.StatusNoContent {
		t.Fatalf("reinicio=%d %s", rec.Code, rec.Body.String())
	}

	// Sessão antiga caiu (qualquer rota autenticada dá 401 com o cookie morto).
	if rec := u.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.50.0.2"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sessão antiga deveria cair, deu %d", rec.Code)
	}
	u2 := novoJar(h)
	rec = u2.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"rita.rh.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.50.0.3")
	if rec.Code != http.StatusOK || corpoJSON(t, rec)["pedeTotp"] != true {
		t.Fatalf("após reinício deveria pedir TOTP: %d %s", rec.Code, rec.Body.String())
	}
	if !auditoriaExiste(t, pool, "totp_reiniciado") {
		t.Fatal("auditoria de reinício ausente")
	}
}

// 2FA opcional desliga; obrigatório não (RF-003, RS-009).
func TestConta_TotpDesativar_OpcionalSim_ObrigatorioNao(t *testing.T) {
	pool, h := sobeConta(t)
	const senha = "SenhaFicticia-Sec-55"
	criarUsuario(t, pool, "Paulo Professor Fictício", "paulo.pro.ficticio@exemplo-escola.test", senha, "PRO")
	fin := novoJar(h)
	const senhaFin = "SenhaFicticia-Fin-56"
	criarUsuario(t, pool, "Nina Fin Fictícia", "nina.fin.ficticia@exemplo-escola.test", senhaFin, "FIN")

	// PRO (opcional): cadastra e desativa com código válido.
	p := novoJar(h)
	p.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"paulo.pro.ficticio@exemplo-escola.test","senha":"`+senha+`"}`, "10.60.0.2")
	rec := p.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.60.0.2")
	segredo := corpoJSON(t, rec)["segredo"].(string)
	codigo, _ := auth.CodigoAtual(segredo)
	p.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"confirmar","codigo":"`+codigo+`"}`, "10.60.0.2")
	codigo2, _ := auth.CodigoAtual(segredo)
	if rec := p.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"desativar","codigo":"`+codigo2+`"}`, "10.60.0.2"); rec.Code != http.StatusOK {
		t.Fatalf("desativar opcional=%d %s", rec.Code, rec.Body.String())
	}

	// FIN (obrigatório): desativar → 403.
	fin.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"nina.fin.ficticia@exemplo-escola.test","senha":"`+senhaFin+`"}`, "10.60.0.3")
	rec = fin.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.60.0.3")
	segredoFin := corpoJSON(t, rec)["segredo"].(string)
	codigoFin, _ := auth.CodigoAtual(segredoFin)
	fin.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"confirmar","codigo":"`+codigoFin+`"}`, "10.60.0.3")
	codigoFin2, _ := auth.CodigoAtual(segredoFin)
	if rec := fin.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"desativar","codigo":"`+codigoFin2+`"}`, "10.60.0.3"); rec.Code != http.StatusForbidden {
		t.Fatalf("desativar obrigatório deveria ser 403, deu %d", rec.Code)
	}
}

// Autorização: anônimo → 401; sem scope → 403; alvo inexistente → 404.
func TestConta_Autorizacao_401_403_404(t *testing.T) {
	pool, h := sobeConta(t)
	const senha = "SenhaFicticia-Sec-71"
	criarUsuario(t, pool, "Lia Sec Fictícia", "lia.sec.ficticia@exemplo-escola.test", senha, "SEC")
	admin := entregaAdmin(t, pool, h)

	anon := novoJar(h)
	if rec := anon.faz(t, http.MethodPost, "/api/conta/senha",
		`{"senhaAtual":"x","senhaNova":"y"}`, "10.70.0.9"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anônimo deveria ser 401, deu %d", rec.Code)
	}
	if rec := anon.faz(t, http.MethodPost, "/api/conta/totp",
		`{"acao":"iniciar"}`, "10.70.0.9"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anônimo totp deveria ser 401, deu %d", rec.Code)
	}

	sec := novoJar(h)
	sec.faz(t, http.MethodPost, "/api/sessao",
		`{"email":"lia.sec.ficticia@exemplo-escola.test","senha":"`+senha+`"}`, "10.70.0.2")
	inexistente := uuid.NewString()
	if rec := sec.faz(t, http.MethodPost, "/api/usuarios/"+inexistente+"/senha-redefinicao", "", "10.70.0.2"); rec.Code != http.StatusForbidden {
		t.Fatalf("SEC em admin deveria ser 403, deu %d", rec.Code)
	}
	if rec := sec.faz(t, http.MethodPost, "/api/usuarios/"+inexistente+"/totp-reinicio", "", "10.70.0.2"); rec.Code != http.StatusForbidden {
		t.Fatalf("SEC em reinício deveria ser 403, deu %d", rec.Code)
	}
	if rec := admin.faz(t, http.MethodPost, "/api/usuarios/"+inexistente+"/senha-redefinicao", "", "10.70.0.1"); rec.Code != http.StatusNotFound {
		t.Fatalf("alvo inexistente deveria ser 404, deu %d", rec.Code)
	}
}

// Rate limit cobre TOTP e redefinição (RS-050/051).
func TestConta_RateLimit_Totp_429(t *testing.T) {
	_, h := sobeConta(t)
	anon := novoJar(h)
	var ultimo int
	for i := 0; i < 6; i++ {
		rec := anon.faz(t, http.MethodPost, "/api/conta/totp", `{"acao":"iniciar"}`, "10.80.0.5")
		ultimo = rec.Code
	}
	if ultimo != http.StatusTooManyRequests {
		t.Fatalf("6ª tentativa deveria ser 429, deu %d", ultimo)
	}
}
