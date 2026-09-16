# ADR-0004 — Hospedagem: Vercel (Hobby) + Neon (Free), com plano B documentado

- **Status:** Aceito (stack definida pelo autor do projeto; não reaberta) — plano B registrado, não implementado
- **Data:** 16/09/2026
- **Relacionados:** PRD §7 (viabilidade gratuita), SRS §3 (restrições), ADR-0001 (backend)

## Contexto

A Diretiva 3 exige custo zero real: open source ou plano gratuito **permanente** — nada de trial, crédito promocional ou plano que exija cartão. O deploy precisa atender: (a) frontend SPA e backend Go no mesmo domínio (cookie first-party, sem CORS, `SameSite=Strict`); (b) dados no Brasil (LGPD — região São Paulo); (c) nenhuma cobrança acidental (plano sem compra de uso extra); (d) banco gerenciado com branch de preview isolada da produção.

## Decisão

- **Vercel, plano Hobby**, um projeto com dois serviços (modelo atual `services` com rewrites — não o antigo `experimentalServices`):
  - serviço **`web`**: SPA Vue como arquivos estáticos, com fallback para `index.html`;
  - serviço **`api`**: backend Go em `/api/*` (Fluid compute), região **`gru1`** (São Paulo) `[VERIFICAR]` se o Hobby permite fixar região.
- **Neon PostgreSQL, plano Free**, região **`aws-sa-east-1`**; string de conexão **com pooler** para a aplicação (pool pequeno por instância) e string **direta** (sem pooler) para migrações goose pelo GitHub Actions.
- **Ambientes isolados:** `DATABASE_URL` de produção só existe no ambiente Production da Vercel e nos secrets do GitHub; previews usam a **branch `preview`** do Neon (dados fictícios, recriável a partir do esquema de produção; o plano Free limita branches).
- **Deploy de produção** pelo GitHub Actions: CI da `main` → migrações com goose → `vercel build` → `vercel deploy --prebuilt --prod`. Deploy automático de produção pela integração Git fica **desligado** (via configuração `git` do `vercel.json` — a confirmar `[VERIFICAR]`).
- **Previews** pela integração Git da Vercel, com proteção de deployment padrão e proteção contra deploy de forks ligadas.
- **Rollback:** instantâneo da Vercel para o código; migrações compatíveis com a versão anterior (expandir → contrair) para o banco.
- **Backups:** `pg_dump` semanal no GitHub Actions, criptografado com **age** (só a chave pública no repositório; a privada fica com o responsável, fora do repo), guardado como artifact com retenção curta — artifacts de repo público podem ser baixados por terceiros. O job também mantém o projeto do Neon ativo.

**Plano B (documentado, não implementado):** se o Vercel Services mudar de forma incompatível, migrar para: backend no **Render** (plano gratuito), SPA em **Cloudflare Worker** com proxy de `/api/*` e cabeçalho de segredo de origem. A migração completa está descrita neste ADR; nenhum código do plano B existe até a necessidade real.

### Passos do plano B (se necessário)

1. Criar serviço web no Render com o mesmo binário Go e `DATABASE_URL` do Neon.
2. Publicar a SPA num Cloudflare Worker (Workers Static Assets) com regra de proxy `/api/*` → Render, exigindo um cabeçalho de segredo de origem (`Origin-Secret`) que só o Worker conhece.
3. Redefinir o domínio/cookie (continua first-party, pois o Worker serve frontend e proxy no mesmo domínio).
4. Atualizar `AGENTS.md`, Taskfile e variáveis de ambiente; arquivar este ADR com o novo status.

## Alternativas consideradas

| Alternativa | Por que não |
| :--- | :--- |
| Render/Cloudflare como escolha principal | Vercel aceita backend Go zero-config e SPA no mesmo projeto/domínio, com régua de custo previsível no Hobby (sem compra de uso extra) |
| Fly.io / Railway | Planos gratuitos mudaram de conditions no passado; exigem cartão ou crédito promocional em algumas modalidades `[VERIFICAR]` |
| Supabase/PlanetScale como banco | Neon foi a escolha da stack (branches por ambiente e região Brasil no Free); não reabrir |
| GitHub Pages para o SPA | Não serve backend no mesmo domínio; quebraria o cookie first-party e exigiria CORS |

## Consequências

- **Positivas:** mesmo domínio para web e API; dados no Brasil; sem risco de cobrança; previews isoladas com dados fictícios; rollback de código instantâneo.
- **Negativas:** Services em beta (risco R-02 do PRD, mitigado pelo plano B); plano Free do Neon limita branches e pode desativar projeto inativo (mitigado pelo backup semanal); limites de consumo precisam de monitoramento nos painéis.
- **Neutras:** o domínio de produção é fornecido pela Vercel (`*.vercel.app`); não há custo de domínio próprio no MVP.
