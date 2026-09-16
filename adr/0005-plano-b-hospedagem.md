# ADR-0005 — Plano B de hospedagem: Render + Cloudflare Workers

- **Status:** Registrado (não implementado) — acionado somente se o Vercel Services mudar de forma incompatível (bloqueio B-02 da Etapa 2)
- **Data:** 16/09/2026
- **Relacionados:** ADR-0004 (hospedagem principal), ETAPA2-VALIDACAO (B-02, I-01, I-02), SRS §16/§18/§29

## Contexto

A hospedagem principal (ADR-0004) usa o modelo `services` da Vercel para colocar SPA e backend Go no **mesmo domínio** — condição que sustenta o cookie first-party, a ausência de CORS e o `SameSite=Strict` (RS-003). O modelo está em beta: pode mudar de forma incompatível. A Diretiva 3 exige custo zero permanente para a alternativa também. Este ADR descreve a migração completa **antes** de qualquer necessidade, para que a execução não fique bloqueada esperando decisão.

## Decisão

Se o Vercel Services se tornar incompatível, migrar para:

- **Backend Go no Render** (plano gratuito): mesmo binário, servidor HTTP padrão `net/http` + chi (ADR-0001), sem acoplamento ao formato serverless da Vercel. Região escolhível nos EUA; `[VERIFICAR]` região/condições correntes do plano gratuito do Render no momento da migração (cartão não exigido é pré-requisito — Diretiva 3).
- **SPA num Cloudflare Worker** (Workers Static Assets, plano gratuito): servir `frontend/dist` e fazer **proxy** de `/api/*` → Render, no **mesmo domínio** do Worker, preservando cookie first-party e sem CORS.
- **Segredo de origem:** o Worker injeta um cabeçalho próprio (ex.: `Origin-Secret`) em toda requisição ao Render; o backend rejeita requisições sem ele, impedindo acesso direto ao domínio do Render pulando o proxy.
- **Banco:** Neon permanece (não muda); migrações e backup seguem pelo GitHub Actions.
- **CSP/headers:** mantidos via configuração do Worker (mesmo conjunto do §18 do SRS).

### Passos da migração (ordem)

1. Provisionar serviço web no Render com o binário Go e a env `DATABASE_URL` (pooler, papel `sacre_app`) + `ORIGIN_SECRET`.
2. Criar o Worker com Static Assets apontando para `frontend/dist` e rota de proxy `/api/*` → domínio do Render com o cabeçalho de segredo.
3. Backend: middleware novo (após o de segurança) que valida `ORIGIN_SECRET`; ligar apenas quando o proxy estiver ativo.
4. Adicionar middleware de "confiança de proxy" para IP real (rate limiting usa IP — RS-050) conforme documentação do Render `[VERIFICAR]`.
5. Atualizar `vercel.json` → arquivar; criar `wrangler` config versionado; Taskfile e `mise.toml` ganham tasks do Worker.
6. Variáveis de ambiente: mesmo conjunto (§26 do SRS), mudando apenas onde vivem (painéis Render/Cloudflare); secrets do GitHub de deploy continuam.
7. Rodar E2E e Lighthouse contra a nova stack local no CI (Playwright não muda; só a orquestração local).
8. Previews: integração do Cloudflare/Render por branch `[VERIFICAR]` suporte no plano gratuito; alternativamente, preview só com stack local no CI.
9. Atualizar AGENTS.md, SRS (§16/§18/§25/§29) e THREAT-MODEL (fronteira 2 passa a ser "Render + Cloudflare"); arquivar este ADR com status "Implementado" e abrir ADR do incidente que o motivou.

### Riscos do plano B

| Risco | Mitigação |
| :--- | :--- |
| Plano gratuito do Render hiberna e responde lento na primeira requisição | Mesmo comportamento já aceito na meta RNF-003 (skeleton + retentativas) |
| Limite de requisições gratuito do Workers | Volume baixo (escola de centenas de alunos — A-03); monitorar nos painéis |
| Segredo de origem no Worker | `ORIGIN_SECRET` nunca no repo (RS-060); rotação documentada |
| Dupla plataforma = mais painéis para revisar | Aceito: é plano B, não o caminho principal |

## Alternativas consideradas

| Alternativa | Por que não |
| :--- | :--- |
| Manter só a Vercel e reverter para formato zero-config sem Services | Se o Services for descontinuado, o mesmo projeto não garante os dois serviços no mesmo domínio; sem garantia, não é plano de contingência |
| Fly.io | Plano gratuito mudou de condições no passado; exigência de cartão em algumas modalidades `[VERIFICAR]` |
| Self-host (VPS gratuito) | Nenhum VPS permanente sem cartão/limites compatíveis com Diretiva 3; operação manual demais para o fluxo por IA |
| Render + Cloudflare Pages sem Worker | Pages não faz proxy de `/api/*` no mesmo domínio com cabeçalho injetado; quebraria o modelo first-party |

## Consequências

- **Positivas:** contingência específica e executável; preserva as propriedades de segurança críticas (same-origin, sem CORS, segredo de origem); custo zero mantido.
- **Negativas:** migração estima-se em 2–3 tarefas de ROADMAP + revisão humana; operação em duas plataformas; preview possivelmente restrita.
- **Neutras:** banco, CI, geração de código e especificação de requisitos não mudam — o impacto fica confinado à fronteira 2 do THREAT-MODEL.
