# ERP Escola — Sacre Cœur des Enfants

ERP escolar open source (monorepo). Especificação em `docs/` (`SRS.md`, `PRD.md`,
`THREAT-MODEL.md`, `ROADMAP.md`, `TUTORIAL.md`). Regras obrigatórias para
qualquer agente ou contribuidor em `AGENTS.md`.

> Estado (F-02): fundação + contrato OpenAPI (`api/openapi.yaml`: saúde e
> sessão, envelope `Erro`, `security` negar-por-padrão) com geração de código
> (oapi-codegen strict-server chi, sqlc, Orval + Zod — gerados nunca editados
> à mão). Sem código de negócio: `task dev` sobe PostgreSQL + backend hello
> (`GET /api/saude`) + frontend hello (Vite); `task verificar` fiscaliza o
> que já existe e endurece junto com cada tarefa do `docs/ROADMAP.md`.

## Pré-requisitos

| Ferramenta | Versão fixada (`mise.toml`) | Instalação |
| :--- | :--- | :--- |
| mise | — | `brew install mise` (ou [mise.jdx.dev](https://mise.jdx.dev)) |
| Go | 1.27.1 | `mise install` na raiz do repo |
| Node | 24.21.0 (LTS) | `mise install` na raiz do repo |
| pnpm | 10.19.0 | `mise install` na raiz do repo |
| Task | ≥ 3.53.1 | `brew install go-task/tap/go-task` |
| oapi-codegen | v2.8.0 | `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0` (F-02) |
| sqlc | v1.31.1 | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1` (gera na F-03) |
| Docker + Compose | — | Docker Desktop ou Colima (`docker compose version`) |
| Gitleaks | — | `brew install gitleaks` (obrigatório: `task verificar` falha sem ele) |

Redocly e Orval vêm do workspace pnpm (`pnpm install` na raiz — versões
fixadas em `package.json`/`frontend/package.json`, ver `mise.toml`). Binários
instalados via `go install` ficam em `$(go env GOPATH)/bin` — garanta esse
diretório no `PATH` para as tasks `gerar:*`.

## Como rodar (`task dev`)

```bash
git clone https://github.com/alienmonk09/erp-escola-opensource.git
cd erp-escola-opensource

mise install        # instala Go, Node e pnpm nas versões fixadas
cp .env.example .env  # credenciais SÓ do banco local (arquivo ignorado — RS-060)
task dev            # PostgreSQL + backend hello + frontend hello, tudo em loopback
```

Depois de subir:

| Serviço | URL local (loopback) |
| :--- | :--- |
| PostgreSQL | `127.0.0.1:5432` (`POSTGRES_PORT`) |
| Backend hello | `http://127.0.0.1:8080/api/saude` (`BACKEND_PORT`) → `{"status":"ok"}` |
| Frontend hello | `http://127.0.0.1:5173/` (`FRONTEND_PORT`, proxy `/api` → backend) |

Banco local (somente desenvolvimento — nunca produção; **sem padrão no
repo**: `POSTGRES_USER` e `POSTGRES_PASSWORD` são obrigatórios e vêm do `.env`
local — AGENTS.md §4.6):

| Variável | Padrão | Uso |
| :--- | :--- | :--- |
| `POSTGRES_USER` | (obrigatório) | usuário do banco local |
| `POSTGRES_PASSWORD` | (obrigatório) | senha **só do banco local** (produção usa secret do Neon) |
| `POSTGRES_DB` | `erp_escola` | database local |
| `POSTGRES_PORT` | `5432` | porta publicada **só em 127.0.0.1** |

Para derrubar o banco: `docker compose down` (os dados ficam no volume
`postgres-data/`; `docker compose down -v` apaga tudo).

## Tasks principais (`Taskfile.yml`, comandos exatos da SRS §19)

| Task | O que faz |
| :--- | :--- |
| `task dev` | `docker compose up -d` + backend Go + Vite (proxy `/api`) |
| `task verificar` | lint + type check + testes + `gerar:checar` + varreduras, em paralelo — **rode antes de todo PR** |
| `task contrato:lint` | valida `api/openapi.yaml` (a partir da F-02) |
| `task gerar:backend` / `gerar:sql` / `gerar:frontend` | regenera código a partir do contrato (nunca edite gerados à mão) |
| `task gerar:checar` | regenera tudo e exige árvore limpa (gate de CI) |
| `task migrar` | `goose up` com `DATABASE_URL_MIGRACAO` (string **direta**, fora do repo) |

## CI

`.github/workflows/ci.yml` (esqueleto em F-01): jobs `backend`, `frontend`,
`contrato` e `segurança` com filtros por caminho (SRS §27), concurrency com
`cancel-in-progress` em PR, cache pnpm/Go e actions fixadas por hash de commit
(RS-061). Sem `pull_request_target` (RS-062).

## Estrutura (SRS §17)

```text
api/                    # contrato OpenAPI (fonte da verdade, a partir da F-02)
backend/cmd/api/        # hello vazio (F-01) → entrypoint real em F-04
backend/internal/       # httpapi|db (GERADOS, F-02/F-03) + auth, autorizacao, auditoria, config, modulos
backend/queries/        # SQL do sqlc (F-03)   backend/migrations/  # goose (F-03)
frontend/               # hello vazio Vite (F-01) → SPA Vue real em F-07 (src/api GERADO)
e2e/                    # Playwright (fases finais)
```

## Segurança (resumo — vale `AGENTS.md`)

- Nenhum segredo no repo (`.env` ignorado; produção via secrets do GitHub/Vercel).
- Só dados fictícios em fixtures/testes — nunca dados reais de aluno/responsável.
- Dependência fora da stack exige ADR e aprovação humana antes.

