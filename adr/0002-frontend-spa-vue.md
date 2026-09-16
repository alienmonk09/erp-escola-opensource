# ADR-0002 — Frontend SPA Vue 3 com dados via TanStack Query

- **Status:** Aceito (stack definida pelo autor do projeto; não reaberta)
- **Data:** 16/09/2026
- **Relacionados:** PRD §1, SRS §3, ADR-0001 (backend), ADR-0003 (segurança)

## Contexto

O frontend do ERP é um sistema de gestão interno com muitas tabelas, formulários e dashboards, usado por perfis com permissões diferentes. Requisitos que pesam na decisão: (a) **nenhum dado sensível persiste no navegador** (Diretiva 1: nada de dados pessoais, tokens ou stores sensíveis em `localStorage`, `sessionStorage` ou IndexedDB); (b) o cache de dados da API precisa ser limpo no logout; (c) o backend fica no **mesmo domínio** (sem CORS, cookie first-party); (d) o código será escrito por IA com contexto limitado, então padrões repetíveis importam mais que flexibilidade.

## Decisão

- **Vue 3 + TypeScript**, SPA pura, sem SSR; **Composition API com `<script setup lang="ts">` sempre** (fiscalizado pelo ESLint).
- **Vite** para build; em desenvolvimento, o servidor do Vite repassa `/api` para o backend Go local.
- **Vue Router** com rotas declaradas em arquivo de configuração, lazy loading e perfis exigidos em `meta` (os guards são UX; a autorização real é no backend).
- **Pinia** apenas para sessão, perfil, preferências (tema) e estado de interface — **não é cache de dados da API**.
- **TanStack Query para Vue** (`@tanstack/vue-query`) como único cache de dados da API, em memória, com retentativas para o primeiro acesso após ociosidade da função ou do banco, e limpeza total no logout.
- **Orval** gera composables do TanStack Query e schemas **Zod** a partir do OpenAPI.
- **PrimeVue** (modo estilizado, tema claro/escuro, locale pt-BR) + **PrimeIcons**; **Tailwind CSS** com plugin **tailwindcss-primeui**; formulários com **PrimeVue Forms** + resolver Zod reusando os schemas gerados; gráficos com o componente **Chart** do PrimeVue.
- Datas e números com a **API `Intl` nativa** (sem biblioteca extra).
- CSP com `script-src 'self'` estrito; `style-src 'unsafe-inline'` é risco aceito por causa dos estilos que o PrimeVue injeta (registrado aqui e no SRS).

## Alternativas consideradas

| Alternativa | Por que não |
| :--- | :--- |
| Nuxt/SSR | Server-side rendering adiciona servidor e estado entre requisições — contraria o modelo serverless da Vercel; SPA basta para sistema interno |
| React/Angular | Stack definida escolheu Vue; não reabrir |
| Vuex ou estado próprio | Pinia é o padrão atual do ecossistema Vue e mais simples para IA |
| Cache da API no Pinia ou localStorage | Violaria Diretiva 1 (persistência sensível no navegador); TanStack Query mantém cache só em memória |
| Biblioteca de datas (dayjs/date-fns) | `Intl` nativo cobre formatação pt-BR; menos dependência (Diretiva 2) |
| Componentes próprios de UI | Custo alto de código gerado pela IA; PrimeVue dá acessibilidade e tema prontos |

## Consequências

- **Positivas:** cache em memória some no logout por construção; schemas Zod únicos servem para formulários e validação; geração por Orval reduz código escrito por IA; tema claro/escuro e pt-BR prontos.
- **Negativas:** `unsafe-inline` em estilos é risco aceito e registrado; bundle do PrimeVue exige disciplina de import seletivo e code splitting para a meta de LCP < 2,5 s.
- **Neutras:** responsividade garantida por breakpoints do Tailwind (RNF-011).
