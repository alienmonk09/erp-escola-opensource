// fundacao_test.go — testes de integração da fundação (F-03).
//
// Requisitos cobertos: migração goose sobe do zero; RS-011 (auditoria
// append-only para `sacre_app`); RS-024 (papéis com privilégio mínimo);
// queries sqlc mínimas (buscar usuário por e-mail, inserir auditoria).
//
// Stack (SRS §28): testcontainers-go contra PostgreSQL real
// (postgres:17.11-alpine, mesma versão do docker-compose.yml).
// Só dados fictícios (RE-06, ver testdata/fixtures_fundacao.sql);
// nenhum segredo em código ou fixture (RS-060/RS-066): o papel `sacre_app`
// é assumido via SET ROLE (sem senha) numa conexão de superusuário
// efêmera do contêiner descartável.
package backend_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	dbgen "github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	// Mesma imagem do docker-compose.yml (RS-061: versão fixada).
	testPostgresImage = "postgres:17.11-alpine"
	// Usuário fictício da fixture versionada (RE-06).
	fixtureEmail = "helena.ficticia@exemplo-escola.test"
	fixtureNome  = "Helena Fictícia da Silva"
)

// sobeBanco retorna o contexto e a string de conexão de um PostgreSQL
// descartável com a migração 0001 aplicada do zero. Pula o teste quando não
// há Docker local (o CI só exige estes testes onde há Docker; gates sem
// Docker continuam passando nas demais tasks do `task verificar`).
func sobeBanco(t *testing.T) (context.Context, string) {
	t.Helper()
	// testcontainers-go fala com o daemon via DOCKER_HOST ou pelo socket
	// padrão; sem nenhum dos dois, pula (ex.: CI sem Docker).
	dockerOK := os.Getenv("DOCKER_HOST") != ""
	for _, sock := range []string{"/var/run/docker.sock", os.Getenv("HOME") + "/.colima/default/docker.sock"} {
		if _, err := os.Stat(sock); err == nil {
			dockerOK = true
		}
	}
	if !dockerOK {
		t.Skip("requer Docker local (testcontainers-go + PostgreSQL real)")
	}
	// Sem Ryuk: o contêiner é descartado pelo Terminate no Cleanup.
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()
	ctr, err := postgres.Run(ctx, testPostgresImage,
		postgres.WithDatabase("erp_escola"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("subir PostgreSQL de teste: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Fatalf("descartar PostgreSQL de teste: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("string de conexão do PostgreSQL de teste: %v", err)
	}

	// Migração goose do zero, como `task migrar` faz no fluxo real (RS-024).
	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("abrir conexão para migrar: %v", err)
	}
	defer sqldb.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialeto goose: %v", err)
	}
	if err := goose.Up(sqldb, "migrations"); err != nil {
		t.Fatalf("goose up do zero: %v", err)
	}

	return ctx, dsn
}

// Tabelas e semente criadas pela migração a partir de banco zerado.
func TestFundacao_MigracaoDoZero(t *testing.T) {
	ctx, dsn := sobeBanco(t)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer conn.Close(ctx)

	for _, tabela := range []string{"usuario", "perfil", "usuario_perfil", "auditoria", "notificacao"} {
		var existe bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`,
			tabela).Scan(&existe)
		if err != nil {
			t.Fatalf("checar tabela %s: %v", tabela, err)
		}
		if !existe {
			t.Fatalf("tabela %s não existe após goose up do zero", tabela)
		}
	}

	var nPerfis int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM perfil`).Scan(&nPerfis); err != nil {
		t.Fatalf("contar perfis: %v", err)
	}
	if nPerfis != 9 {
		t.Fatalf("semente de perfis: esperado 9, obtido %d", nPerfis)
	}
	for _, codigo := range []string{"SEC", "FIN", "COO", "CPR", "RH", "DIR", "ADM", "PRO", "RES"} {
		var existe bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM perfil WHERE codigo = $1)`, codigo).Scan(&existe); err != nil {
			t.Fatalf("checar perfil %s: %v", codigo, err)
		}
		if !existe {
			t.Fatalf("perfil %s ausente na semente", codigo)
		}
	}
}

// RS-011: `auditoria` é append-only para `sacre_app` (INSERT/SELECT ok;
// UPDATE/DELETE/TRUNCATE negados). O papel é assumido via SET ROLE, sem
// senha e sem segredo em código (RS-060).
func TestFundacao_AuditoriaAppendOnly(t *testing.T) {
	ctx, dsn := sobeBanco(t)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer conn.Close(ctx)

	// Linha-base inserida como dono, para as tentativas de escrita do papel app.
	var auditoriaID int64
	err = conn.QueryRow(ctx,
		`INSERT INTO auditoria (acao, recurso, detalhe) VALUES ('teste_base', 'teste:1', '{}') RETURNING id`,
	).Scan(&auditoriaID)
	if err != nil {
		t.Fatalf("inserir auditoria-base: %v", err)
	}

	// Superusuário efêmero vira membro do papel para poder assumi-lo via SET ROLE.
	if _, err := conn.Exec(ctx, `GRANT sacre_app TO postgres`); err != nil {
		t.Fatalf("assumir papel sacre_app: %v", err)
	}
	if _, err := conn.Exec(ctx, `SET ROLE sacre_app`); err != nil {
		t.Fatalf("SET ROLE sacre_app: %v", err)
	}
	// Defer (LIFO: executa antes do conn.Close registrado acima).
	defer func() {
		if _, err := conn.Exec(context.Background(), `RESET ROLE`); err != nil {
			t.Errorf("RESET ROLE: %v", err)
		}
	}()

	// INSERT e SELECT: permitidos (RS-024: DML necessário).
	var inseridoID int64
	err = conn.QueryRow(ctx,
		`INSERT INTO auditoria (acao, recurso, detalhe) VALUES ('login', 'usuario:teste', '{}') RETURNING id`,
	).Scan(&inseridoID)
	if err != nil {
		t.Fatalf("sacre_app deveria conseguir INSERT em auditoria: %v", err)
	}
	var lidos int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM auditoria`).Scan(&lidos); err != nil {
		t.Fatalf("sacre_app deveria conseguir SELECT em auditoria: %v", err)
	}

	// UPDATE, DELETE e TRUNCATE: negados (RS-011).
	for nome, stmt := range map[string]string{
		"UPDATE":   `UPDATE auditoria SET acao = 'adulterado' WHERE id = ` + strconv.FormatInt(auditoriaID, 10),
		"DELETE":   `DELETE FROM auditoria WHERE id = ` + strconv.FormatInt(auditoriaID, 10),
		"TRUNCATE": `TRUNCATE auditoria`,
	} {
		if _, err := conn.Exec(ctx, stmt); err == nil {
			t.Fatalf("sacre_app conseguiu %s em auditoria (RS-011 violado)", nome)
		} else if !strings.Contains(strings.ToLower(err.Error()), "permission denied") {
			t.Fatalf("%s em auditoria falhou sem ser por permissão: %v", nome, err)
		}
	}

	// Sanidade do privilégio mínimo (RS-024): UPDATE em tabela comum ok.
	if _, err := conn.Exec(ctx, `UPDATE perfil SET nome = nome WHERE codigo = 'SEC'`); err != nil {
		t.Fatalf("sacre_app deveria conseguir UPDATE em perfil: %v", err)
	}
}

// Queries sqlc mínimas contra dados fictícios versionados (RE-06).
func TestFundacao_QueriesSqlc(t *testing.T) {
	ctx, dsn := sobeBanco(t)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer conn.Close(ctx)

	fixture, err := os.ReadFile(filepath.Join("testdata", "fixtures_fundacao.sql"))
	if err != nil {
		t.Fatalf("ler fixture: %v", err)
	}
	if _, err := conn.Exec(ctx, string(fixture)); err != nil {
		t.Fatalf("aplicar fixture: %v", err)
	}

	q := dbgen.New(conn)

	usuario, err := q.GetUsuarioPorEmail(ctx, fixtureEmail)
	if err != nil {
		t.Fatalf("GetUsuarioPorEmail: %v", err)
	}
	if usuario.Nome != fixtureNome || !usuario.Ativo {
		t.Fatalf("usuário fictício divergente: %+v", usuario)
	}

	if _, err := q.GetUsuarioPorEmail(ctx, "ninguem@exemplo-escola.test"); err != pgx.ErrNoRows {
		t.Fatalf("e-mail inexistente deveria dar ErrNoRows, deu: %v", err)
	}

	registro, err := q.InsertAuditoria(ctx, dbgen.InsertAuditoriaParams{
		UsuarioID: usuario.ID,
		Acao:      "login",
		Recurso:   "usuario:11111111-1111-4111-8111-111111111111",
		Detalhe:   []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("InsertAuditoria: %v", err)
	}
	if registro.ID == 0 || registro.Acao != "login" {
		t.Fatalf("auditoria inserida divergente: %+v", registro)
	}

	var nNotif int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM notificacao WHERE usuario_id = $1 AND lida_em IS NULL`,
		usuario.ID).Scan(&nNotif); err != nil {
		t.Fatalf("contar notificações: %v", err)
	}
	if nNotif != 1 {
		t.Fatalf("notificações abertas do usuário fictício: esperado 1, obtido %d", nNotif)
	}
}
