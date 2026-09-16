# ADR-0001 — Backend em Go com contrato OpenAPI e SQL gerado

- **Status:** Aceito (stack definida pelo autor do projeto; não reaberta)
- **Data:** 16/09/2026
- **Relacionados:** PRD §1, SRS §3, ADR-0002 (hospedagem)

## Contexto

O ERP precisa de um backend que: (a) rode na Vercel em plano gratuito com cobrança apenas por CPU ativa; (b) seja verificável por máquinas — compilador forte, type check e lint — porque o código será escrito por modelos de IA open source com contexto limitado (Diretiva 4); (c) lide com dinheiro (mensalidades, folha) sem erro de arredondamento; (d) tenha contrato de API como primeira barreira de validação de entrada (Diretiva 1).

## Decisão

Backend em **Go (mínimo 1.25)** no formato zero-config de backend Go da Vercel, com:

- **chi v5** como roteador HTTP (compatível com `net/http` e suportado pelo oapi-codegen);
- **OpenAPI 3 escrito primeiro**; código do servidor gerado por **oapi-codegen** em modo *strict server* para chi;
- validação de requisições pelo middleware de validação do oapi-codegen (baseado em kin-openapi);
- lint do contrato com **Redocly CLI**;
- acesso a dados com **sqlc + pgx v5**: SQL escrito à mão, código Go gerado — somente consultas parametrizadas;
- **dinheiro em centavos, como inteiro**, em BRL (nunca ponto flutuante);
- datas/horas em UTC, exibidas no fuso `America/Fortaleza`;
- `log/slog` (biblioteca padrão) em JSON; configuração com **caarlos0/env**.

## Alternativas consideradas

| Alternativa | Por que não |
| :--- | :--- |
| Node.js/TypeScript no backend | Stack definida usa TypeScript no frontend; duplicar runtime aumenta superfície de erro da IA e abandona o compilador forte de Go |
| Python (FastAPI/Django) | Menos verificação estática que Go; cold start e empacotamento em serverless menos previsíveis para a Vercel |
| Java/Kotlin | Binários e memória maiores para serverless gratuito; build mais lento no CI |
| ORM (GORM, Prisma) | Diretiva 2/4: SQL explícito gerado por sqlc é verificável e evita mágica; menos abstração para a IA errar |
| Escrever validação à mão | O contrato OpenAPI gerando validação é determinístico e cobre o "negar por padrão" na fronteira |

## Consequências

- **Positivas:** binário único e rápido; `http.CrossOriginProtection` (Go ≥ 1.25) nativo para CSRF; geração determinística reduz o que a IA escreve à mão; centavos como inteiro eliminam erro de ponto flutuante.
- **Negativas:** curva de aprendizado maior para estudantes iniciantes (mitigado pelo docs/TUTORIAL.md); sqlc exige disciplina de escrever SQL; formato zero-config da Vercel precisa ser confirmado na documentação no momento da implementação `[VERIFICAR]`.
- **Neutras:** migrações com goose rodam pelo GitHub Actions com papel de banco próprio, nunca pela aplicação.
