// orval.config.ts — cliente frontend gerado a partir do contrato (F-02)
// Fonte: SRS §19. Gera em `frontend/src/api/` (NUNCA editar à mão — AGENTS.md
// §4, SRS §17):
// - `erp.ts` (+ `model/`): composables TanStack Query (vue-query sobre fetch)
//   + tipos TypeScript;
// - `erp.zod.ts`: schemas Zod para validação nos formulários (SRS §22).
// Uso: `task gerar:frontend` (= `pnpm --filter frontend exec orval`).
// Versão fixada: orval 8.33.0 (ver mise.toml). Dependências de runtime
// (@tanstack/vue-query, zod v4 — versão fixada em `override.zod.version`)
// chegam na F-07 com o shell do frontend; até lá o gerado não é compilado.
import { defineConfig } from 'orval';

export default defineConfig({
  erp: {
    input: {
      target: '../api/openapi.yaml',
    },
    output: {
      mode: 'single',
      client: 'vue-query',
      target: './src/api/erp.ts',
      schemas: './src/api/model',
      override: {
        // TanStack Query v5 (major corrente; F-07 instala @tanstack/vue-query
        // v5 — sem isso o orval gera hooks no formato v4).
        query: {
          version: 5,
        },
      },
    },
  },
  erpZod: {
    input: {
      target: '../api/openapi.yaml',
    },
    output: {
      mode: 'single',
      client: 'zod',
      target: './src/api/erp.zod.ts',
      override: {
        zod: {
          version: 4,
        },
      },
    },
  },
});
