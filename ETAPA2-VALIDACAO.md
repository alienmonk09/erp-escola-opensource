# Etapa 2 — Relatório de validação

> **Status da Etapa 1:** concluída — `PRD.md`, `SRS.md`, `THREAT-MODEL.md`, `adr/0001`–`0004`, `TUTORIAL.md` (capítulos 1–4 e seção de lições).
> **Esta etapa não reabre a stack.** Apresenta apenas: bloqueios, incompatibilidades e confirmação do escopo do MVP.
> **Aviso:** a busca na web estava indisponível durante esta execução, então **nenhum limite de plano foi confirmado ao vivo**. Tudo que não pôde ser confirmado está marcado `[VERIFICAR]` e consolidado no fim, para checagem rápida antes de iniciar a Etapa 3.

---

## 1. Bloqueios

**Nenhum bloqueio confirmado.** Nenhum item da stack apresentou evidência de mudança de plano gratuito, licença ou manutenção que viole as diretivas — porém, por falta de confirmação ao vivo, os seguintes itens **travam o início da implementação** até checagem (você pode checá-los nos painéis/docs em ~30 minutos):

| # | Item | O que checar | Se falhar, alternativa mais próxima |
| :--- | :--- | :--- | :--- |
| B-01 | Vercel Hobby × região `gru1` | Se o plano Hobby permite fixar a região das funções em São Paulo (`[VERIFICAR]`) | Sem fixação de região: aceitar região padrão (provavelmente `iad1`/EUA) — impacto: latência maior e dados fora do Brasil, o que **violaria RE-05/LGPD (RP-011)**. Nesse caso, o plano B (ADR-0004: Render + Cloudflare, com região escolhível) sai do papel — decisão sua |
| B-02 | Vercel Services (beta) | Se o modelo `services` + rewrites está disponível e estável; se `vercel build` + `vercel deploy --prebuilt` funciona com projetos que usam Services (`[VERIFICAR]`) | Plano B do ADR-0004 (Render backend + Cloudflare Worker servindo a SPA com proxy `/api/*` e segredo de origem) |
| B-03 | Neon Free | CU-horas, nº de branches (precisamos de 2: produção + preview), projeto inativo 90 dias, região `aws-sa-east-1` (`[VERIFICAR]`) | Se branches < 2: previews sem banco (mock da API) ou CI local; se região indisponível: mesmo dilema de residência de dados do B-01 |
| B-04 | GitHub Actions | Minutos ilimitados em runners padrão para repos públicos; storage/retensão de artifacts (backup age semanal) (`[VERIFICAR]`) | Reduzir frequência (backup mensal), artifacts menores, ou backup fora do Actions (manual com cron local) |
| B-05 | Agentes de IA | Catálogo gratuito do OpenCode Zen no dia da execução; alternativa Freebuff/Kilo Code (`[VERIFICAR]`) | A regra é fixa (modelo open source gratuito, preferindo sem treino sobre dados) — trocar de ferramenta/modelo não bloqueia, pois as tarefas não dependem de recursos exclusivos |

**Conclusão sobre bloqueios:** o único bloqueio potencialmente estrutural é o **B-01/B-03 (residência de dados no Brasil nos planos gratuitos)**. Se ambos falharem, as opções honestas são: (a) ativar o plano B documentado no ADR-0004, ou (b) registrar em ADR a exceção (dados fora do Brasil) como risco aceito para o ambiente **acadêmico com dados fictícios** — jamais para operação real com dados de alunos.

## 2. Incompatibilidades identificadas

Avaliei os cruzamentos pedidos pelo prompt (formato zero-config de backend Go × Vercel Services × SPA Vue; fluxo `--prebuilt` × Services; Hobby × região) e mais os que notei:

| # | Cruzamento | Risco | Como resolvemos na especificação |
| :--- | :--- | :--- | :--- |
| I-01 | **Formato zero-config de backend Go × Services + rewrites** | O formato zero-config define entrypoint/porta por convenção; com Services, é preciso confirmar como o serviço `api` aponta para o backend e como o rewrite `/api/*` chega a ele | A Etapa 3 fixa o `vercel.json` a partir da documentação corrente (`[VERIFICAR]`); o SRS (RF/RS) independe desse detalhe, e o ROADMAP isolou isso numa tarefa de fundação com critério de aceite verificável (health checks respondendo via rewrite) |
| I-02 | **`vercel build` + `vercel deploy --prebuilt` × Services** | Pode haver diferença entre o deploy pela integração Git (preview) e o deploy prebuilt (produção via Actions) em projetos com Services | A arquitetura já usa os dois caminhos separados por design (preview via Git; produção só via Actions). Tarefa de fundação valida os dois antes de qualquer feature. Se incompatível: produção passa pela integração Git da `main` com checks obrigatórios (perde a ordem migração→deploy; registrar em ADR) — `[VERIFICAR]` |
| I-03 | **Hobby × região das funções** | Ver B-01 | B-01 |
| I-04 | **Neon pooler × pgx** | O pooler do Neon (PgBouncer em modo transacional) não suporta prepared statements por sessão; pgx por padrão usa cache de statements | A stack já prevê "configuração do pgx compatível com o pooler"; a Etapa 3 fixa: conexão via pooler com `statement_cache_mode` desativado/ajustado e pool pequeno por instância; migrações usam a string **direta** (sem pooler) — já definido na stack. `[VERIFICAR]` nome exato do parâmetro na versão do pgx v5 corrente |
| I-05 | **Cookie `__Host-` × preview da Vercel** | Cookies `__Host-` exigem HTTPS e domínio sem subdomínio? Não: exigem Secure, path=/, sem Domain — funcionam em `*.vercel.app`; porém cada preview tem subdomínio próprio, e o cookie é por host, o que é aceitável | Sem mudança de design; teste de fundação cobre login num preview |
| I-06 | **CSRF `http.CrossOriginProtection` × Orval/TanStack** | A proteção nativa valida *origin/sec-fetch*; o cliente HTTP gerado pelo Orval envia requisições same-origin do navegador, então passa naturalmente; chamadas server-to-server (se houver) precisarão de token próprio | Nenhuma chamada server-to-server no MVP; critério de aceite da fundação inclui mutação passando pela proteção |
| I-07 | **CSP estrita × PrimeVue (`style-src 'unsafe-inline'`)** | Risco aceito e registrado (ADR-0002); `script-src 'self'` permanece estrito | Já tratado |
| I-08 | **Rate limiting no PostgreSQL × cold start** | O contador no PG adiciona uma consulta por requisição; após ociosidade, isso soma ao cold start | Limites aplicados só a rotas sensíveis (login, mutações críticas) na Etapa 3; leituras gerais sem rate limit por PG (apenas firewall da Vercel, se disponível) |
| I-09 | **Backup age via Actions × repositório público** | Artifact baixável por terceiros | Já mitigado por design: criptografia age obrigatória (só chave pública no repo) — RS-064 |
| I-10 | **TOTP obrigatório × criação do primeiro admin** | O primeiro admin precisa cadastrar TOTP sem estar logado | Fluxo: login com senha temporária → troca obrigatória de senha → cadastro TOTP guiado antes de liberar o painel. Tarefa de fundação cobre isso com critérios Given/When/Then |

**Nenhuma incompatibilidade é impeditiva.** As I-01/I-02/I-03 dependem de confirmação documental no início da Etapa 3 e têm plano B registrado.

## 3. Confirmação do escopo do MVP

Reproduzo o escopo proposto na Etapa 1 (PRD §3) para sua confirmação:

**Dentro do MVP (Must/Should):**

1. **Transversal:** login (senha + TOTP obrigatório p/ Direção, Financeiro, RH, Admin; opcional p/ demais), RBAC negar-por-padrão, auditoria append-only, notificações internas, criação segura do primeiro admin, redefinição de senha por admin, recuperação de TOTP.
2. **Acadêmico:** cadastro único de alunos/responsáveis, matrícula/rematrícula (gera contrato + mensalidades), documentos (metadados), histórico/declarações em PDF, consulta da situação financeira.
3. **Pedagógico:** turmas, alocação de professores, notas, frequência, boletim em PDF, situação final.
4. **Financeiro:** mensalidades geradas pela matrícula, baixa de pagamento, boleto demonstrativo, inadimplência, contas a pagar (fornecedores + folha).
5. **Compras/Estoque:** requisições com consulta de saldo, fornecedores, pedidos com aprovação, entradas/saídas, alertas de mínimo.
6. **RH:** funcionários/professores, contratos, carga horária, folha por competência.
7. **Gerencial:** painéis agregados (inadimplência, alunos, gastos, estoque) + exportação auditada.
8. **Portal do responsável (Should, somente leitura):** situação financeira, boletim, avisos — só do próprio filho.

**Fora do MVP (Won't):** integração bancária real, e-mail, pagamento online, app nativo, multi-escola, biblioteca/diário de classe/ocorrências, integrações governamentais, portal LGPD de autoatendimento. (Justificativas no PRD §3.2.)

**Assunções que preciso que você valide** (PRD §6.1): A-01 (portal só leitura), A-02 (navegador moderno, sem app), A-03 (escola de centenas de alunos — base das metas de performance), A-04 (boleto demonstrativo), A-05 (só dados fictícios), A-06 (um ano letivo ativo).

## 4. Pendências de verificação `[VERIFICAR]` (checklist da Etapa 2)

1. Vercel Hobby: fixar região `gru1` das funções.
2. Vercel Services: disponibilidade/estabilidade do modelo `services` + rewrites; compatibilidade com `vercel build` + `vercel deploy --prebuilt`.
3. Desligar deploy automático de produção da integração Git via `vercel.json` (chave exata da configuração `git`).
4. Convenção de entrypoint/porta do formato zero-config de backend Go.
5. Neon Free: CU-horas, nº de branches, inatividade de 90 dias, região `aws-sa-east-1`.
6. Vercel Hobby: proteção de deployment, proteção contra deploy de forks, limite de builds de preview/dia, firewall (regras de rate limit extra).
7. GitHub Actions: minutos em runners padrão (repo público), storage e retenção de artifacts.
8. GitHub: secret scanning e push protection em repos públicos.
9. pgx v5: parâmetro exato de compatibilidade com pooler (modo de cache de statements).
10. OpenCode Zen: catálogo gratuito atual (e alternativas Freebuff/Kilo Code).
11. Versões correntes: ASVS, OWASP Top 10 (Web/API), Go 1.25.x (`http.CrossOriginProtection`), bibliotecas da stack e suas licenças.
12. Prazo legal de guarda de documentos escolares (RP-005).

---

**Resolução:** escopo e suposições A-01…A-06 aprovados em 16/09/2026; pendências mantidas com `[VERIFICAR]` para checagem na fundação. **Etapa 3 concluída:** especificação fechada em SRS §16–§31, `AGENTS.md` e `adr/0005-plano-b-hospedagem.md`.
