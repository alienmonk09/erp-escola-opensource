# ERP Escola — Sacre Cœur des Enfants

ERP escolar open source (monorepo). Especificação em `docs/` (`SRS.md`, `PRD.md`,
`THREAT-MODEL.md`, `ROADMAP.md`, `TUTORIAL.md`). Regras obrigatórias para
qualquer agente ou contribuidor em `AGENTS.md`.

> Estado (F-01): fundação — toolchain e ambiente local. Ainda sem código de
> negócio: `task dev` sobe o PostgreSQL e os serviços que já existirem
> (backend/frontend chegam em F-04/F-07); `task verificar` passa na árvore vazia
> e endurece junto com cada tarefa do `docs/ROADMAP.md`.

## Pré-requisitos

| Ferramenta | Versão fixada (`mise.toml`) | Instalação |
| :--- | :--- | :--- |
| mise | — | `brew install mise` (ou [mise.jdx.dev](https://mise.jdx.dev)) |
| Go | 1.27.1 | `mise install` na raiz do repo |
| Node | 24.21.0 (LTS) | `mise install` na raiz do repo |
| pnpm | 10.19.0 | `mise install` na raiz do repo |
| Task | ≥ 3.53.1 | `brew install go-task/tap/go-task` |
| Docker + Compose | — | Docker Desktop ou Colima (`docker compose version`) |

## Como rodar (`task dev`)

```bash
git clone https://github.com/alienmonk09/erp-escola-opensource.git
cd erp-escola-opensource

mise install        # instala Go, Node e pnpm nas versões fixadas
task dev            # sobe o PostgreSQL local (localhost:5432) + backend/frontend quando existirem
```

Banco local (somente desenvolvimento — nunca produção):

| Variável | Padrão local | Uso |
| :--- | :--- | :--- |
| `POSTGRES_USER` | `sacre` | usuário do banco local |
| `POSTGRES_PASSWORD` | `sacre_local_dev_only` | senha **só do banco local** (fora do repo; produção usa secret do Neon) |
| `POSTGRES_DB` | `erp_escola` | database local |
| `POSTGRES_PORT` | `5432` | porta publicada |

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
backend/cmd/api/        # entrypoint (F-04)
backend/internal/       # httpapi|db (GERADOS) + auth, autorizacao, auditoria, config, modulos
backend/queries/        # SQL do sqlc (F-03)   backend/migrations/  # goose (F-03)
frontend/src/           # api (GERADO) + modules, components, stores, router, lib (F-07)
e2e/                    # Playwright (fases finais)
```

## Segurança (resumo — vale `AGENTS.md`)

- Nenhum segredo no repo (`.env` ignorado; produção via secrets do GitHub/Vercel).
- Só dados fictícios em fixtures/testes — nunca dados reais de aluno/responsável.
- Dependência fora da stack exige ADR e aprovação humana antes.
