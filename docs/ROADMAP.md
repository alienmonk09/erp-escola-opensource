# ROADMAP — ERP Escola Sacre Cœur des Enfants (Etapa 4)

> **Etapa 4 — Plano para execução por IA.** Primeira fase é a **fundação**; nenhuma funcionalidade de negócio começa antes dela.
> Fontes de verdade: `PRD.md`, `SRS.md` (§1–§31), `THREAT-MODEL.md`, `ETAPA2-VALIDACAO.md`, `../AGENTS.md`, `../adr/0001`–`0005`.
> Convenção: **um PR = uma tarefa**. PR pequeno (revisão ≤ 30 min). Título em Conventional Commits com ID da tarefa.
> Verificação local obrigatória antes de todo PR: `task verificar`. CI verde é gate (RS-060..062, RE-07).
> Dados sempre fictícios (RE-06, RP-008). Nenhum segredo no repo (RS-060/066). Código gerado nunca editado à mão (SRS §19).

## Como usar

1. Execute as tarefas **em ordem** dentro de cada fase; respeite **Dependências**.
2. Cada tarefa lista **Requisitos atendidos** (IDs). Leia esses itens no SRS/PRD/THREAT-MODEL antes de codar — a especificação é a verdade.
3. Cole o **Prompt pronto** no OpenCode (ou Freebuff/Kilo Code). Registre no corpo do PR: **ID da tarefa, agente e modelo usados, requisitos atendidos, o que mudou, riscos, como testar** (RS-067).
4. **Nenhum PR revisado pelo mesmo modelo que o escreveu** (Diretiva 4).
5. Itens `[VERIFICAR]` (SRS §31) são resolvidos na fundação (F-01, F-09); se um `[VERIFICAR]` falhar, registre ADR e siga o plano B (ADR-0005), nunca invente.

## Mapa de fases

| Fase | Objetivo | Tarefas |
| :--- | :--- | :--- |
| **0 — Fundação** | Repo, toolchain, contrato inicial, banco, auth, TOTP, RBAC, auditoria, deploy hello autenticado, CI/gates, frontend base | F-01 … F-10 |
| **1 — Acadêmico + Financeiro base** | Alunos, matrículas (gera mensalidades), baixa, boleto, inadimplência | N-01 … N-04 |
| **2 — Pedagógico** | Turmas, notas/frequência, boletim PDF | P-01 … P-03 |
| **3 — Compras/Estoque** | Requisições, fornecedores, pedidos, movimentos, alertas | C-01 … C-03 |
| **4 — RH + Financeiro contas a pagar** | Funcionários, folha, contas a pagar | R-01 … R-02 |
| **5 — Gerencial + Portal + Notificações** | Painéis, exportação auditada, portal responsável, avisos | G-01 … G-03 |
| **6 — Endurecimento e go-live** | E2E completo, Lighthouse, ZAP, backup restore, docs finais | E-01 … E-02 |

Total: 24 tarefas. Dependência entre fases: Fase N só começa com a anterior mesclada na `main` e CI verde.

---

## Fase 0 — Fundação (obrigatória primeiro)

### F-01 — Repositório, toolchain e ambiente local

- **Status:** 🟡 em revisão — PR #1 (https://github.com/alienmonk09/erp-escola-opensource/pull/1), 2026-09-16.
- **Dependências:** nenhuma (primeira tarefa).
- **Requisitos atendidos:** SRS §17, §19, §27; RNF-005; RS-061.
- **Contexto mínimo:** Monorepo da SRS §17. `mise.toml` fixa Go (≥1.25), Node, pnpm. `docker-compose.yml` sobe PostgreSQL local. `Taskfile.yml` tem as tasks do §19 (`contrato:lint`, `gerar:*`, `gerar:checar`, `dev`, `migrar`, `verificar`). Nenhum código de negócio aqui.
- **Bibliotecas a usar:** nenhuma nova (toolchain: mise, Task, Docker Compose).
- **Arquivos a criar ou alterar:** `mise.toml`, `Taskfile.yml`, `docker-compose.yml`, `.github/workflows/ci.yml` (esqueleto com filtros por caminho + concurrency), `.github/dependabot.yml`, `.gitignore`, `README.md` (como rodar `task dev`).
- **Instruções passo a passo:**
  1. Criar estrutura de pastas vazia da §17 (sem handlers ainda).
  2. `mise.toml` com versões fixas; documentar que flags de CLI são `[VERIFICAR]` (SRS §31 item 11) e registrar as versões escolhidas.
  3. `docker-compose.yml` com PostgreSQL para dev/testes.
  4. `Taskfile.yml` com as 8 tasks da §19 (comandos exatos; `verificar` roda lint+typecheck+testes+gerar:checar em paralelo).
  5. CI esqueleto: jobs separados backend/frontend/contrato/segurança, filtros por caminho (§27), `cancel-in-progress: true`, cache pnpm/Go.
  6. Dependabot (Go, npm, Actions).
- **Cuidados de segurança:** Actions fixadas por **hash de commit** (RS-061); sem `pull_request_target` (RS-062); sem segredos neste PR.
- **Critérios de aceite verificáveis:**
  - Given repo clonado, When `task dev` roda, Then PostgreSQL local sobe e backend/frontend sobem (mesmo que com hello vazio).
  - Given qualquer PR, When CI roda, Then jobs filtram por caminho e `task verificar` passa localmente.
- **Comando de verificação local:** `task verificar`
- **Pré-requisito manual (humano, uma única vez antes da F-01):** criar o repositório **público** na **conta pessoal** do GitHub (o plano Hobby da Vercel não conecta repos de organização, e os minutos ilimitados do Actions valem para repo público — PRD §6.3), conectá-lo e publicar a `main`:
  1. `gh repo create <repo> --public` (ou em `github.com/new`: público, sem README/gitignore/licença — o conteúdo já existe localmente).
  2. `git remote add origin https://github.com/<conta>/<repo>.git && git push -u origin main`.
  3. Ativar **Secret scanning** e **Push protection** em *Settings → Code security* (TUTORIAL cap. 5.1).
- **Passos manuais (humano):** o pré-requisito acima; nada além disso.
- **Prompt pronto:**
  ```text
  Tarefa F-01 (Fundação: repo + toolchain). Leia SRS §17, §19, §27 e AGENTS.md §6-7.
  Crie mise.toml, Taskfile.yml, docker-compose.yml, CI esqueleto com filtros por caminho e Dependabot.
  Não adicione dependência fora da stack. Não crie código de negócio. Rode `task verificar` e abra o PR `chore(F-01): toolchain e ambiente local`.
  ```

### F-02 — Contrato OpenAPI inicial + geração de código

- **Status:** 🟡 em revisão — PR #7 (https://github.com/alienmonk09/erp-escola-opensource/pull/7), 2026-09-17.
- **Dependências:** F-01.
- **Requisitos atendidos:** SRS §12, §15; RS-020, RS-022, RS-030, RS-035; RF-001, RF-002, RF-008.
- **Contexto mínimo:** Contrato-first (AGENTS.md §6). `api/openapi.yaml` é a fonte da verdade. Escopo deste PR: `GET /api/saude`, `GET /api/saude/pronto`, `POST /api/sessao`, `POST /api/sessao/totp`, `DELETE /api/sessao` + envelope `Erro` (§15.3) + `security: sessaoCookie` negar-por-padrão (§15.1). Convenções §15.2 (UUID, centavos, datas, paginação, CPF dígitos, enums).
- **Bibliotecas a usar:** oapi-codegen (strict-server, chi), Redocly CLI, sqlc (config vazia), Orval.
- **Arquivos a criar ou alterar:** `api/openapi.yaml`, `api/redocly.yaml` (regras: operação sem security = erro; erro sem schema = erro), `backend/oapi-cfg.yaml`, `backend/sqlc.yaml`, `frontend/orval.config.ts`. Gerados: `backend/internal/httpapi/`, `backend/internal/db/`, `frontend/src/api/` (via tasks, nunca à mão).
- **Instruções passo a passo:**
  1. Escrever `openapi.yaml` só com as 5 operações acima + schemas `Erro`, `PaginaDe` (convenção, mesmo que ainda sem uso).
  2. `task contrato:lint` verde.
  3. Rodar `task gerar:backend`, `task gerar:sql`, `task gerar:frontend`; commitar gerados.
  4. `task gerar:checar` verde (gate).
- **Cuidados de segurança:** toda operação herda `security: sessaoCookie` exceto as 3 públicas listadas (§12); mutações exigem CSRF (anotar, implementar em F-04).
- **Critérios de aceite:**
  - Given `api/openapi.yaml` alterado, When CI roda, Then Redocly passa e `gerar:checar` confirma gerados em dia.
  - Given payload fora do schema, When validado, Then 400 com envelope `Erro` (teste de contrato em F-04).
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum.
- **Prompt pronto:**
  ```text
  Tarefa F-02 (Contrato inicial). Leia SRS §12, §15 e AGENTS.md §6. Crie api/openapi.yaml só com saude/sessao + envelope Erro + security negar-por-padrão. Configure Redocly, oapi-codegen, sqlc, Orval. Rode task gerar:* e task gerar:checar. PR `feat(F-02): contrato OpenAPI inicial e geração`.
  ```

### F-03 — Banco: papéis, migrações base, auditoria append-only

- **Status:** 🟡 em revisão — PR #8 (https://github.com/alienmonk09/erp-escola-opensource/pull/8), 2026-09-17.
- **Dependências:** F-01.
- **Requisitos atendidos:** SRS §20 (§20.1, §20.2, §20.7, §20.8); RS-010, RS-011, RS-024; RN-002 (parcial: usuario/perfil); RE-06.
- **Contexto mínimo:** Dois papéis: `sacre_migracao` (dono, só Actions via string direta) e `sacre_app` (só SELECT/INSERT/UPDATE, sem UPDATE/DELETE em `auditoria`). Migrações goose em `backend/migrations/` (expandir→contrair). Tabelas desta tarefa: `usuario`, `perfil`, `usuario_perfil`, `auditoria`, `notificacao` (§20.2). Testes com testcontainers-go contra PostgreSQL real.
- **Bibliotecas a usar:** goose, sqlc + pgx v5, testcontainers-go.
- **Arquivos a criar ou alterar:** `backend/migrations/0001_fundacao.sql`, `backend/queries/fundacao.sql` (queries mínimas: buscar usuário por email, inserir auditoria), `backend/*_test.go` de integração.
- **Instruções passo a passo:**
  1. Migração cria papéis + 5 tabelas + `REVOKE UPDATE,DELETE,TRUNCATE ON auditoria FROM sacre_app`.
  2. Queries sqlc para as operações acima; `task gerar:sql`.
  3. Teste: tentativa de UPDATE/DELETE em auditoria com `sacre_app` falha (RS-011); migração sobe do zero.
  4. Fixtures fictícias versionadas (RE-06).
- **Cuidados de segurança:** string direta só em `DATABASE_URL_MIGRACAO` (GH secret, nunca no repo); sem seed de usuário/senha (RS-004).
- **Critérios de aceite:**
  - Given banco zerado, When `task migrar` roda, Then tabelas existem e auditoria é append-only para `sacre_app`.
  - Given teste de integração, When roda, Then passa com PostgreSQL real.
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum (Neon vem em F-06/F-09).
- **Prompt pronto:**
  ```text
  Tarefa F-03 (Banco fundação). Leia SRS §20.1, §20.2 e AGENTS.md §7. Crie migração goose com usuario/perfil/usuario_perfil/auditoria/notificacao + papéis sacre_migracao/sacre_app + REVOKE na auditoria. Queries sqlc mínimas + testes testcontainers (append-only). Fixtures fictícias. PR `feat(F-03): banco fundação e auditoria append-only`.
  ```

### F-04 — Auth: sessão, RBAC, rate limiting, headers, health

- **Status:** 🟡 em revisão.

- **Dependências:** F-02, F-03.
- **Requisitos atendidos:** SRS §9, §12, §25; RF-001, RF-002, RF-008; RS-001, RS-003, RS-005, RS-006, RS-010, RS-012, RS-020..023, RS-040, RS-050, RS-051; AM-001, AM-002.
- **Contexto mínimo:** Backend Go + chi. Sessão scs + pgxstore, cookie `__Host-` (HttpOnly, Secure, SameSite=Strict), inatividade 30 min / absoluto 12 h, renova ID no login (RS-005), troca de senha encerra todas as sessões (RS-006). Middleware único de autorização: scopes do OpenAPI → perfis no banco (§9), negar por padrão. Rate limiting httprate com contador no PG (login 5/min/IP + bloqueio progressivo RS-012). `http.CrossOriginProtection` nas mutações. Headers unrolled/secure. Health `/api/saude` (sem banco) e `/api/saude/pronto` (com banco). Auditoria de login/falha sem dados pessoais (RS-010/066). Hash argon2id (alexedwards/argon2id).
- **Bibliotecas a usar:** chi, scs + pgxstore, argon2id, httprate, unrolled/secure, caarlos0/env, log/slog, oapi-codegen strict-server.
- **Arquivos a criar ou alterar:** `backend/cmd/api/main.go`, `backend/internal/config/`, `backend/internal/auth/`, `backend/internal/autorizacao/`, `backend/internal/auditoria/`, `backend/internal/modulos/admin/` (só sessão aqui), `backend/queries/` + testes.
- **Instruções passo a passo:**
  1. Implementar handlers strict-server só para as 5 rotas de F-02 (login ainda sem TOTP — TOTP em F-05; login devolve "pede TOTP" quando aplicável).
  2. Middlewares na ordem: segurança → rate limit → sessão → autorização → auditoria.
  3. Testes de autorização da matriz §9 para estas rotas: anônimo→401, sem scope→403 (RS-023).
  4. Logs slog JSON sem dados pessoais.
- **Cuidados de segurança:** mensagem genérica de credenciais inválidas (RS-001/013); sem `localStorage`; pooler com `statement_cache_mode` ajustado `[VERIFICAR]` (SRS §31 item 9).
- **Critérios de aceite:**
  - Given senha errada, When login, Then 401 genérico + auditoria `login_falha`.
  - Given sem sessão, When chama rota autenticada, Then 401; Given sem scope, Then 403.
  - Given 6 logins/min do mesmo IP, When tenta, Then 429.
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum.
- **Prompt pronto:**
  ```text
  Tarefa F-04 (Auth base). Leia SRS §9, §12, §25, §11.1 e THREAT-MODEL §3.1. Implemente sessão scs+pgxstore, login argon2id, middlewares RBAC/rate-limit/CSRF/headers, health checks, auditoria de login. Testes de autorização 401/403 + rate limit. PR `feat(F-04): sessão, RBAC e headers`.
  ```

### F-05 — TOTP obrigatório/opcional, troca de senha, redefinição e recuperação

- **Dependências:** F-04.
- **Requisitos atendidos:** RF-004, RF-005, RF-007; RS-004, RS-006..009; US-001, US-004, US-005; AM-003, AM-011; THREAT-MODEL §3.1.
- **Contexto mínimo:** TOTP pquerna/otp, segredo cifrado com secretbox (`TOTP_ENCRYPTION_KEY`, 32 bytes, só Production). Obrigatório para FIN/RH/DIR/ADM, opcional para demais (§9). Fluxos: primeiro login com senha temporária → troca obrigatória → cadastro TOTP guiado (I-10); redefinição por admin gera senha temporária de uso único + troca obrigatória + encerra sessões; recuperação de TOTP por admin com revalidação + auditoria.
- **Bibliotecas a usar:** pquerna/otp, x/crypto/nacl/secretbox, argon2id, scs.
- **Arquivos a criar ou alterar:** `backend/internal/auth/totp.go`, handlers `POST /api/conta/totp`, `POST /api/conta/senha`, `POST /api/usuarios/{id}/senha-redefinicao`, `POST /api/usuarios/{id}/totp-reinicio`, `api/openapi.yaml` (adicionar operações + scopes `admin:usuarios:gerenciar`), regenerar.
- **Instruções passo a passo:**
  1. Atualizar contrato primeiro; `task contrato:lint` + `gerar:*`.
  2. Implementar os 4 endpoints + auditoria (RS-010).
  3. Testes: perfil obrigatório sem TOTP → bloqueado; troca de senha encerra todas as sessões; senha temporária exige troca.
- **Cuidados de segurança:** segredo TOTP nunca em log (RS-066); códigos TOTP nunca persistidos; rate limit em TOTP e redefinição.
- **Critérios de aceite:**
  - Given FIN sem TOTP, When login só com senha, Then bloqueado até cadastrar TOTP.
  - Given redefinição por admin, When usuário troca a senha, Then sessões antigas invalidadas + auditoria.
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum.
- **Prompt pronto:**
  ```text
  Tarefa F-05 (TOTP + senha). Leia SRS §9, §12, §25, US-001/004/005. Atualize openapi.yaml (conta/totp, conta/senha, redefinição, totp-reinicio), regenere, implemente com secretbox + auditoria + testes. PR `feat(F-05): TOTP e gestão de senha`.
  ```

### F-06 — Vercel (2 serviços) + Neon (2 branches) + primeiro admin + hello autenticado

- **Dependências:** F-04, F-05.
- **Requisitos atendidos:** SRS §16, §18, §26, §29; RS-004, RS-060, RS-063; US-003; I-01, I-02, I-10; B-01..B-03.
- **Contexto mínimo:** `vercel.json` da §18 (services web+api, rewrites `/api/*`, fallback SPA, headers, `region: gru1`, `git.deploymentEnabled.main: false` — chaves `[VERIFICAR]`). Deploy produção só via Actions: `goose up` (direta, expandir) → `vercel build` → `vercel deploy --prebuilt --prod`. Preview via integração Git + branch `preview` do Neon (dados fictícios). Primeiro admin via comando administrativo único no fluxo (senha de secret, troca + TOTP no primeiro login).
- **Bibliotecas a usar:** nenhuma nova (Vercel CLI, goose).
- **Arquivos a criar ou alterar:** `vercel.json`, `.github/workflows/deploy.yml`, `backend/cmd/admin/main.go` (cria admin idempotente), `frontend/` hello autenticado mínimo (placar de sessão).
- **Instruções passo a passo:**
  1. Escrever `vercel.json` conforme §18; registrar chaves confirmadas ou `[VERIFICAR]` resolvidos em F-09.
  2. Workflow deploy com as 3 etapas na ordem + comando admin (idempotente, recusa se já existe admin).
  3. Provar: health checks respondem via rewrite; login funciona num preview (I-05, I-06).
- **Cuidados de segurança:** `DATABASE_URL` produção só em Production + GH secrets; preview nunca vê produção (RS-063); proteção contra deploy de forks ligada.
- **Critérios de aceite:**
  - Given merge na main com CI verde, When deploy roda, Then migração→build→deploy e `/api/saude` responde via `/api/*`.
  - Given preview, When login, Then funciona contra branch `preview` com dados fictícios.
  - Given admin já existe, When comando roda de novo, Then recusa (idempotente) e não há credencial padrão (RS-004).
- **Comando de verificação local:** `task verificar`
- **Passos manuais (humano):** criar contas GitHub/Vercel/Neon (TUTORIAL cap. 5); fixar região `gru1`; criar branches `main`/`preview`; cadastrar envs/secrets da §26. Marcar como tarefa do humano no PR.
- **Prompt pronto:**
  ```text
  Tarefa F-06 (Deploy + admin). Leia SRS §16, §18, §26, §29 e ADR-0004. Crie vercel.json, workflow deploy (migrate→build→prebuilt) e comando admin idempotente. Prove health via rewrite + login em preview. PR `feat(F-06): deploy Vercel+Neon e primeiro admin`.
  ```

### F-07 — Frontend base: tema, router, stores, query, cliente gerado

- **Dependências:** F-02, F-06.
- **Requisitos atendidos:** SRS §21, §22, §23, §13; RNF-002, RNF-003, RNF-006, RNF-007; RE-12; RS-033.
- **Contexto mínimo:** Vue 3 + TS, sempre `<script setup lang="ts">`. PrimeVue styled mode (preset Aura `[VERIFICAR]`), locale pt-BR, Tailwind + tailwindcss-primeui. Pinia só sessão/tema/ui; TanStack Query para dados (§22). Orval gera composables + Zod. Chaves §23.1, staleTime §23.2, invalidações §23.3, retry 5xx + 401 global + `queryClient.clear()` no logout (§23.4). Wrappers `frontend/src/components/` (§21). Sem `v-html`, sem `localStorage` exceto tema. Rotas com `meta.perfis` + lazy loading (guards só UX).
- **Bibliotecas a usar:** Vue Router, Pinia, TanStack Query, Orval+Zod, PrimeVue+PrimeIcons, Tailwind, Intl nativo.
- **Arquivos a criar ou alterar:** `frontend/src/router/`, `frontend/src/stores/`, `frontend/src/components/` (7 wrappers), `frontend/src/lib/`, páginas `Login`, `Inicio`, `vite.config.ts` (proxy `/api`), `frontend/` lint (ESLint vue+ts, Prettier, vue-tsc).
- **Instruções passo a passo:**
  1. Configurar tema claro/escuro + `MoneyText`, `DateText` (Intl), `PageSkeleton`.
  2. Router com login + início + guard de perfil.
  3. Stores sessão/tema/ui + QueryClient com retry/logout da §23.4.
  4. Tela de login usa mutação gerada pelo Orval; erro inline via Zod.
- **Cuidados de segurança:** backend revalida tudo (RS-021); 401 limpa cache e volta ao login.
- **Critérios de aceite:**
  - Given login válido, When entra, Then menu filtra por perfil (UX) e refresh revalida.
  - Given logout, When sai, Then `queryClient.clear()` e nenhum dado da API permanece.
  - Given build, When Lighthouse local, Then sem regressão grosseira de bundle.
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum.
- **Prompt pronto:**
  ```text
  Tarefa F-07 (Frontend base). Leia SRS §13, §21, §22, §23. Monte router+Pinia+Query+Orval, wrappers do design system, login e início, tema claro/escuro, guards só UX. Respeite RE-12 e RS-033. PR `feat(F-07): shell do frontend e design system base`.
  ```

### F-08 — CI completo com gates + backup criptografado

- **Dependências:** F-01, F-04, F-07.
- **Requisitos atendidos:** SRS §27, §25; RS-060..062, RS-064, RS-065; RNF-005; B-04.
- **Contexto mínimo:** Jobs paralelos backend/frontend/contrato/segurança (Gitleaks, OSV-Scanner, zizmor, govulncheck, CodeQL agendado). E2E/Lighthouse/ZAP só na main/agendado. Backup semanal `pg_dump` + age (só chave pública no repo) como artifact com retenção curta; job mantém Neon ativo (B-03).
- **Bibliotecas a usar:** nenhuma nova (Actions, age).
- **Arquivos a criar ou alterar:** `.github/workflows/ci.yml`, `deploy.yml`, `backup.yml`, `codeql.yml`, `zap.yml`, `Taskfile.yml` (tasks de varredura local).
- **Instruções passo a passo:**
  1. Completar workflows da §27 com hashes de commit e sem `pull_request_target`.
  2. Backup com `AGE_PUBLIC_KEY` no repo; documentar guarda da privada fora do repo.
  3. Provar restore uma vez em ambiente descartável (não produção).
- **Cuidados de segurança:** artifact público => criptografia obrigatória (RS-064); secrets nunca em PR de fork.
- **Critérios de aceite:**
  - Given PR com segredo fake, When CI roda, Then Gitleaks falha e merge bloqueia.
  - Given agendamento, When backup roda, Then artifact criptografado é gerado e restore funciona em banco descartável.
- **Comando de verificação local:** `task verificar`
- **Passos manuais (humano):** gerar par de chaves age; guardar privada fora do repo; cadastrar pública.
- **Prompt pronto:**
  ```text
  Tarefa F-08 (CI gates + backup). Leia SRS §25, §27 e AGENTS.md §9. Complete workflows com Gitleaks/OSV/zizmor/govulncheck/CodeQL e backup pg_dump+age. Prove bloqueio por segredo e restore em banco descartável. PR `feat(F-08): gates de segurança e backup`.
  ```

### F-09 — Checagem dos `[VERIFICAR]` da fundação (humano + agente)

- **Dependências:** F-06, F-08.
- **Requisitos atendidos:** SRS §31 (itens 1–15); ETAPA2-VALIDACAO §4; B-01..B-05; I-01..I-10.
- **Contexto mínimo:** Nenhum limite foi confirmado ao vivo. Esta tarefa resolve os 15 itens na documentação/painéis e registra o resultado em ADR ou na §31. Se B-01/B-03 falharem (dados fora do Brasil), acionar ADR-0005 ou registrar exceção para ambiente acadêmico fictício (nunca para operação real).
- **Bibliotecas a usar:** nenhuma.
- **Arquivos a criar ou alterar:** `SRS.md` §31 (marcar confirmados), `../adr/0006-verificacoes-fundacao.md` (novo, se alguma decisão mudar), `vercel.json`/`mise.toml` (só se a doc exigir chave diferente).
- **Instruções passo a passo (humano executa, agente registra):**
  1. Vercel: região `gru1` no Hobby, bloco `services`, `vercel build`+`--prebuilt` com Services, chave `git.deploymentEnabled`, proteção de deployment/forks, limite previews/dia, firewall.
  2. Neon: CU-horas, branches, inatividade 90 dias, `aws-sa-east-1`, conexão da branch preview.
  3. GitHub: minutos públicos, artifacts, secret scanning/push protection.
  4. pgx pooler, OpenCode Zen, versões (ASVS, Top 10, Go 1.25, libs+licenças, CLIs), PrimeVue preset, prazo guarda documentos, ASVS do backup, `pg_trgm` no Neon.
- **Cuidados de segurança:** sem segredo no registro; LGPD residência (RE-05/RP-011).
- **Critérios de aceite:** cada um dos 15 itens marcado como confirmado (com link/versão) ou com ADR de contingência; `task gerar:checar` continua verde.
- **Comando de verificação local:** `task verificar`
- **Passos manuais (humano):** todos os checks nos painéis/docs (~30 min).
- **Prompt pronto:**
  ```text
  Tarefa F-09 (VERIFICAR). Leia SRS §31 e ETAPA2-VALIDACAO §4. Para cada um dos 15 itens, registre a confirmação (painel/doc/versão) ou abra ADR de contingência. Não invente limites. PR `docs(F-09): verificação dos pendentes da fundação`.
  ```

### F-10 — E2E da fundação: login, TOTP, RBAC, auditoria

- **Dependências:** F-04..F-08.
- **Requisitos atendidos:** SRS §28; US-001..005; RS-023; RNF-006; AM-001..003, AM-010, AM-011.
- **Contexto mínimo:** Playwright contra stack local no CI (nunca produção/preview protegida) + axe-core. Fluxos: login+TOTP (D1), troca de senha, redefinição por admin, tentativa sem permissão → 403, auditoria confere.
- **Bibliotecas a usar:** Playwright, @axe-core/playwright, Vitest.
- **Arquivos a criar ou alterar:** `e2e/fundacao.spec.ts`, `e2e/axe.spec.ts`, fixtures fictícias.
- **Instruções passo a passo:**
  1. Subir stack local no CI; rodar E2E + axe em claro e escuro.
  2. Cobrir matriz de autorização destas rotas (tabela gerada da §9).
- **Cuidados de segurança:** E2E nunca aponta para produção; sem dados reais.
- **Critérios de aceite:** E2E verde na main; violação AA bloqueia; 401/403 comportam-se como §12.
- **Comando de verificação local:** `task verificar`
- **Passos manuais:** nenhum.
- **Prompt pronto:**
  ```text
  Tarefa F-10 (E2E fundação). Leia SRS §28 e §11.1. Crie E2E Playwright (login/TOTP/RBAC/auditoria) + axe claro/escuro contra stack local. PR `test(F-10): E2E da fundação`.
  ```

**Gate da fundação:** só avance para a Fase 1 com F-01..F-10 mescladas, `task verificar` verde, preview de login funcionando e §31 resolvida ou com ADR.

---

## Fase 1 — Acadêmico + Financeiro base

### N-01 — Alunos e responsáveis (cadastro único)

- **Dependências:** F-10.
- **Requisitos atendidos:** US-101; RF-101..103; RN-002; RS-020, RS-030, RS-032; RP-002, RP-003; AM-012.
- **Contexto mínimo:** `POST/GET /api/alunos`, `GET/PUT /api/alunos/{id}`, vínculo `aluno_responsavel` (base da posse do portal). CPF só dígitos + brdoc no backend; CPF duplicado → 409 (RN-002). Scope `academico:alunos:*`. Frontend: páginas Alunos + FormDialog com Zod do Orval; chave `["alunos",…]`.
- **Bibliotecas:** brdoc, sqlc/pgx, Orval/Zod, DataPage.
- **Arquivos:** `api/openapi.yaml`, `backend/queries/alunos.sql`, `backend/migrations/0002_academico.sql`, `backend/internal/modulos/academico/`, `frontend/src/modules/alunos/`.
- **Passo a passo:** contrato → gerar → migração → queries → handlers → auditoria → telas → invalidação.
- **Segurança:** teste de autorização (SEC escreve; FIN/COO só leem o permitido; anônimo 401); sem CPF em log.
- **Aceite:** Given CPF duplicado, When cria, Then 409; Given FIN, When tenta criar aluno, Then 403; mutação invalida `["alunos"]`.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa N-01 (Alunos). Leia US-101, RF-101..103, RN-002, §23.3. Contrato→gerar→migração→handlers→telas DataPage/FormDialog. Testes 401/403/409. PR feat(N-01): cadastro único de alunos.`

### N-02 — Matrícula e rematrícula (gera contrato + mensalidades)

- **Dependências:** N-01.
- **Requisitos atendidos:** US-102, US-105; RF-104..106; RN-001, RN-003; RS-010, RS-020; AM-004; FX-01.
- **Contexto mínimo:** Transação única: matrícula + contrato + mensalidades em centavos (nunca float). Uma ativa por aluno/ano (constraint). Cancelamento antes das aulas pode cancelar pendentes; paga nunca cancela (estorno auditado). Invalida `matriculas`, `alunos`, `paineis/alunos`.
- **Bibliotecas:** sqlc (transação), maroto (nada aqui ainda).
- **Arquivos:** `api/openapi.yaml` (`/api/matriculas`), migração `0003_matriculas.sql`, handlers, telas Matrículas.
- **Segurança:** scope `academico:matriculas:escrever`; auditoria `matricula_efetivada`; idempotência (dupla submissão → 409).
- **Aceite:** Given aluno cadastrado, When matricula, Then contrato+mensalidades criados; Given segunda ativa no mesmo ano, Then 409. E2E FX-01.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa N-02 (Matrícula). Leia US-102/105, RN-001/003, §11.2. Contrato→migração→transação→tela→E2E FX-01. PR feat(N-02): matrícula gera mensalidades.`

### N-03 — Baixa de pagamento + situação financeira

- **Dependências:** N-02.
- **Requisitos atendidos:** US-103, US-301, US-302; RF-109, RF-301, RF-303; RN-010; RS-010, RS-011, RS-020; AM-006; FX-02; RNF-007/013 (≤60 s).
- **Contexto mínimo:** `POST /api/titulos/{id}/baixa` imutável (correção = ajuste novo, nunca UPDATE). Baixa atualiza situação visível à Secretaria em ≤60 s via invalidação (§23.3). Auditoria `baixa_registrada`.
- **Arquivos:** migração títulos/baixas, queries, handlers, telas Financeiro + situação do aluno.
- **Segurança:** scope `financeiro:baixas:escrever`; teste 403 para SEC; append-only.
- **Aceite:** Given título pendente, When baixa, Then situação "em dia" e `titulos/inadimplencia/paineis/aluno-situacao/portal` invalidados. E2E FX-02.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa N-03 (Baixa). Leia US-302, RN-010, §11.3, §23.3. Baixa imutável + auditoria + invalidação + E2E FX-02. PR feat(N-03): baixa de pagamento.`

### N-04 — Boleto demonstrativo + inadimplência (+ documentos Should)

- **Dependências:** N-03.
- **Requisitos atendidos:** US-104 (Should), US-303, US-304; RF-107, RF-304, RF-305; RP-002, RP-003; AM-006.
- **Contexto mínimo:** PDFs maroto v2 gerados na requisição, streaming direto, sem disco (RE-08). Boleto sem registro bancário. Inadimplência por aluno/turma/escola (agregação no banco). Documentos: metadados + auditoria.
- **Bibliotecas:** maroto v2.
- **Arquivos:** handlers boleto/inadimplência/documentos, telas correspondentes.
- **Segurança:** scopes `financeiro:boletos:ler`, `financeiro:inadimplencia:ler`, `academico:documentos:*`; exportação auditada (RN-011).
- **Aceite:** Given título pendente, When emite boleto, Then PDF volta na resposta e auditoria registra; Given vencidos, When consulta inadimplência, Then agrega por turma/escola.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa N-04 (Boleto+inadimplência). Leia US-303/304/104. PDFs maroto sob demanda + agregações no banco + auditoria. PR feat(N-04): boleto e inadimplência.`

---

## Fase 2 — Pedagógico

### P-01 — Turmas e alocação (só matriculados)

- **Dependências:** N-02.
- **Requisitos:** US-201; RF-201, RF-202; RN-004; RS-020; AM-007.
- **Contexto:** `POST /api/turmas`, `POST /api/turmas/{id}/alunos`; RN-004 validada no banco + backend. Chaves `["turmas"]`, `["turma",id,"alunos"]`.
- **Aceite:** aluno sem matrícula ativa → 409; professor alocado via RH (fluxo em P-03/R-01).
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa P-01 (Turmas). Leia US-201, RN-004. Contrato→migração→handlers→telas. Testes 403/409. PR feat(P-01): turmas.`

### P-02 — Notas e frequência (com auditoria de alteração)

- **Dependências:** P-01.
- **Requisitos:** US-202; RF-204..206; RN-005, RN-008; RS-010, RS-020, RS-023; RP-003; AM-005; FX-03.
- **Contexto:** Professor só nas próprias turmas (RN-005 via `TURMA_PROFESSOR`). Alteração de nota exige `pedagogico:notas:alterar` e grava valor anterior (sem dado pessoal no log). Invalida notas/frequência/boletim/portal.
- **Aceite:** Given professor de outra turma, When lança nota, Then 403; Given nota alterada, When consulta auditoria, Then há registro com autor/data/anterior. E2E FX-03.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa P-02 (Notas). Leia US-202, RN-005/008, §11.4. Lançamento+alteração auditada + E2E. PR feat(P-02): notas e frequência.`

### P-03 — Boletim PDF + necessidade de professores

- **Dependências:** P-02.
- **Requisitos:** US-203, US-204 (Should), US-503 (Should); RF-207, RF-203, RF-209, RF-702; RP-003, RP-004, RP-006; RN-008.
- **Contexto:** Boletim maroto sob demanda; responsável só vê o próprio filho (posse via `aluno_responsavel`). Tela de necessidade por turma (lê `rh:funcionarios:ler` só dados profissionais).
- **Aceite:** Given responsável de outro aluno, When pede boletim, Then 403; Given notas lançadas, When gera, Then PDF + auditoria.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa P-03 (Boletim). Leia US-203/204, RN-008. PDF sob demanda + posse + E2E. PR feat(P-03): boletim e grade.`

---

## Fase 3 — Compras e Estoque

### C-01 — Requisições com consulta de saldo

- **Dependências:** F-10 (qualquer setor abre).
- **Requisitos:** US-401, US-402; RF-401, RF-402; RN-006; RS-020; AM-008; FX-04 (início).
- **Contexto:** `POST /api/requisicoes-material`; avaliação indica "atender do estoque" vs "sugerir compra da diferença". Invalida `requisicoes/estoque/alertas/pedidos`.
- **Aceite:** Given saldo suficiente, When avalia, Then sem compra; Given saldo insuficiente, Then sugere diferença.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa C-01 (Requisições). Leia US-401/402, RN-006. Handlers+telas+invalidação. PR feat(C-01): requisições com saldo.`

### C-02 — Fornecedores e pedidos com aprovação

- **Dependências:** C-01.
- **Requisitos:** US-403; RF-403..405; RN-007, RN-009; RS-010, RS-020; RP-004; AM-008.
- **Contexto:** Pedido aprovado + NF gera conta a pagar (RN-007). Acima do valor parâmetro exige DIR (RN-009, scope `compras:pedidos:aprov` + DIR).
- **Aceite:** Given pedido acima do limite sem DIR, When aprova, Then 403; Given aprovado com NF, Then conta a pagar criada.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa C-02 (Pedidos). Leia US-403, RN-007/009. Aprovação + conta a pagar + auditoria. PR feat(C-02): fornecedores e pedidos.`

### C-03 — Entrada/saída e alertas de mínimo

- **Dependências:** C-02.
- **Requisitos:** US-404; RF-406..408; RN-006; RS-010, RS-020; AM-008; FX-04 (fim).
- **Contexto:** `POST /api/estoque/movimentos`; saída abaixo do mínimo gera notificação interna. Invalida `estoque/alertas/paineis`.
- **Aceite:** Given saída que zera abaixo do mínimo, When registra, Then alerta criado. E2E FX-04 completo.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa C-03 (Estoque). Leia US-404, RN-006. Movimentos + alertas + E2E FX-04. PR feat(C-03): estoque e reposição.`

---

## Fase 4 — RH + Financeiro contas a pagar

### R-01 — Funcionários, contratos e carga horária

- **Dependências:** P-01 (para alocação).
- **Requisitos:** US-501; RF-501, RF-502; RS-020; RP-004; AM-009.
- **Contexto:** `rh:funcionarios:*`; dados remuneratórios com acesso restrito. Cadastro aparece para Coordenação alocar (fluxo RH→Coordenação).
- **Aceite:** Given COO, When lista funcionários, Then só dados profissionais; Given RH, When cadastra, Then auditoria.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa R-01 (RH base). Leia US-501, RP-004. Cadastro+contratos+carga + posse. PR feat(R-01): funcionários.`

### R-02 — Folha por competência + contas a pagar (fornecedor + folha)

- **Dependências:** R-01, C-02.
- **Requisitos:** US-502, US-305; RF-503, RF-504, RF-306..308; RN-007, RN-012; RS-010, RS-020; RP-004; AM-008, AM-009; FX-05.
- **Contexto:** Folha uma vez por competência (reenvio exige cancelamento auditado). Financeiro paga fornecedor/folha (`financeiro:pagamentos:escrever`). Valores em centavos.
- **Aceite:** Given folha enviada, When reenvia mesma competência, Then 409; Given pagamento, When confirma, Then auditoria + fluxo de caixa. E2E FX-05.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa R-02 (Folha+pagar). Leia US-502/305, RN-012. Folha+contas a pagar + E2E. PR feat(R-02): folha e pagamentos.`

---

## Fase 5 — Gerencial + Portal + Notificações

### G-01 — Painéis agregados (4 indicadores)

- **Dependências:** N-04, C-03, R-02.
- **Requisitos:** US-601..604; RF-601..604; RS-020; RP-003; AM-006.
- **Contexto:** `GET /api/paineis/{indicador}` com agregação no banco (nunca no cliente). Chart.js via PrimeVue Chart. Chaves `["paineis",…]`, stale 60 s.
- **Aceite:** Given dados nos módulos, When abre painel, Then inadimplência/alunos/gastos/estoque conferem com as listagens.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa G-01 (Painéis). Leia US-601..604. Agregações sqlc + Chart + cache 60s. PR feat(G-01): painéis gerenciais.`

### G-02 — Exportação auditada em PDF

- **Dependências:** G-01.
- **Requisitos:** US-605; RF-605; RN-011; RS-010, RS-020; AM-006.
- **Contexto:** `GET /api/paineis/{indicador}/relatorio` gera PDF sob demanda + auditoria `exportacao_dados` (quem/o quê/quando, sem dados extras).
- **Aceite:** Given exportação, When consulta auditoria (ADM/DIR), Then registro existe.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa G-02 (Exportação). Leia US-605, RN-011. PDF maroto + auditoria. PR feat(G-02): exportação auditada.`

### G-03 — Portal do responsável + notificações internas

- **Dependências:** N-03, P-03.
- **Requisitos:** US-701, US-702 (Should), US-703 (Could); RF-701..703; RS-020, RS-023; RP-003, RP-006; AM-006, AM-012; FX-06.
- **Contexto:** `portal:consulta` + posse (`aluno_responsavel`): só o próprio filho. Somente leitura. `GET /api/notificacoes` + `PUT` marcar lida; avisos revalidam com aba visível (RNF-007).
- **Aceite:** Given RES de outro aluno, When consulta, Then 403; Given aviso publicado, When RES consulta, Then aparece. E2E FX-06.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa G-03 (Portal). Leia US-701..703, RF-701..703. Posse + somente leitura + E2E. PR feat(G-03): portal do responsável.`

---

## Fase 6 — Endurecimento e go-live

### E-01 — E2E ponta a ponta + acessibilidade + performance

- **Dependências:** todas as Fases 1–5.
- **Requisitos:** SRS §28; RNF-002, RNF-006; RS-023.
- **Contexto:** Playwright cobre FX-01..FX-06 de ponta a ponta contra stack local; axe em claro/escuro sem violação AA; Lighthouse CI com LCP/INP/CLS + bundle ≤250 kB gzip.
- **Aceite:** E2E verde na main; Lighthouse dentro de RNF-002; axe sem violação.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa E-01 (E2E final). Leia SRS §28. E2E FX-01..06 + axe + Lighthouse. PR test(E-01): E2E e performance finais.`

### E-02 — ZAP baseline, restore de backup e docs finais

- **Dependências:** E-01.
- **Requisitos:** SRS §25, §27; RS-064; RP-005 (`[VERIFICAR]` prazo guarda).
- **Contexto:** ZAP baseline agendado contra stack local (baseline versionado, exceções justificadas). Provar restore do backup age em banco descartável. Atualizar TUTORIAL (lições) e matriz §14 se requisito mudou.
- **Aceite:** ZAP sem achado alto sem justificativa; restore funciona; `task verificar` verde.
- **Verificação:** `task verificar`
- **Prompt pronto:** `Tarefa E-02 (Go-live). Leia SRS §25/§27. ZAP + restore + docs. PR chore(E-02): endurecimento final.`

---

## Riscos e plano B

- Se qualquer `[VERIFICAR]` da fundação falhar (região, Services, branches), aplicar ADR-0005 antes de N-01.
- Se previews estourarem cota do Hobby, restringir previews a PRs de frontend (SRS §29).
- Todo endpoint novo exige teste de autorização (AGENTS.md §5); ação sensível exige auditoria (RS-010).

## Pendências de verificação

Nenhuma nova. As 15 da SRS §31 são resolvidas em F-09.
