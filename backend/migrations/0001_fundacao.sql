-- 0001_fundacao.sql — Banco fundação: papéis, tabelas transversais, auditoria append-only.
--
-- Tarefa F-03 (ROADMAP). Requisitos: SRS §20 (§20.1, §20.2, §20.7, §20.8);
-- RS-010, RS-011, RS-024; RN-002 (parcial: usuario/perfil); RE-06.
--
-- Convenções (SRS §20): snake_case; PKs uuid (gen_random_uuid()), exceto
-- `auditoria.id` que é BIGSERIAL explícito (§20.2); timestamptz UTC (RE-09);
-- FKs ON DELETE RESTRICT por padrão.
--
-- Papéis (RS-024):
--   * `sacre_migracao`: dono do esquema; usado SÓ pelas migrações goose via
--     string direta (DATABASE_URL_MIGRACAO, secret fora do repo).
--   * `sacre_app`: usado pela aplicação (pgx, via pooler); só o DML necessário
--     (SELECT/INSERT/UPDATE), sem DDL, sem UPDATE/DELETE/TRUNCATE em auditoria.
-- SEGURANÇA (RS-004/RS-060): esta migração cria os papéis SEM senha (NOLOGIN).
-- O LOGIN + senha de cada papel é provisionado FORA do repo (Neon/painel e
-- GitHub Secrets/Vercel). Nenhum seed de usuário/senha aqui.
--
-- Auditoria append-only (RS-010/RS-011): REVOKE UPDATE, DELETE, TRUNCATE ON
-- auditoria FROM sacre_app (testado em backend/fundacao_test.go).
--
-- Migração expandir→contrair (AGENTS.md §7): só cria; nada destrutivo no Up.
-- Ferramentas verificadas: goose v3.24.3, sqlc v1.29.0, postgres:17.11-alpine.

-- +goose Up

-- gen_random_uuid() (PG17: via extensão pgcrypto).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Papéis idempotentes (re-execução segura; sem senha — ver cabeçalho).
-- StatementBegin/End: o bloco DO contém ";" internos e precisa ir ao banco
-- como uma única instrução (goose divide o arquivo em ";" por padrão).
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'sacre_migracao') THEN
    CREATE ROLE sacre_migracao NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'sacre_app') THEN
    CREATE ROLE sacre_app NOLOGIN;
  END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE usuario (
  id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  nome                 text        NOT NULL,
  email                text        NOT NULL UNIQUE,
  hash_senha           text        NOT NULL,
  senha_temporaria     boolean     NOT NULL DEFAULT false,
  troca_obrigatoria    boolean     NOT NULL DEFAULT false,
  totp_secreto_cifrado bytea       NULL,
  totp_ativo           boolean     NOT NULL DEFAULT false,
  ativo                boolean     NOT NULL DEFAULT true,
  criado_em            timestamptz NOT NULL DEFAULT now(),
  atualizado_em        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE perfil (
  id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  codigo    text NOT NULL UNIQUE,
  nome      text NOT NULL
);

CREATE TABLE usuario_perfil (
  usuario_id uuid NOT NULL REFERENCES usuario (id) ON DELETE RESTRICT,
  perfil_id  uuid NOT NULL REFERENCES perfil (id) ON DELETE RESTRICT,
  PRIMARY KEY (usuario_id, perfil_id)
);
-- Sem linha = sem permissão (negar por padrão, SRS §20.2).
CREATE INDEX idx_usuario_perfil_perfil ON usuario_perfil (perfil_id);

CREATE TABLE auditoria (
  id         bigserial   PRIMARY KEY,
  usuario_id uuid        NULL REFERENCES usuario (id) ON DELETE RESTRICT,
  acao       text        NOT NULL,
  recurso    text        NOT NULL,
  detalhe    jsonb       NOT NULL DEFAULT '{}',
  ip         text        NULL,
  criado_em  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notificacao (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  usuario_id uuid        NOT NULL REFERENCES usuario (id) ON DELETE RESTRICT,
  titulo     text        NOT NULL,
  mensagem   text        NOT NULL,
  lida_em    timestamptz NULL,
  criado_em  timestamptz NOT NULL DEFAULT now()
);

-- Índices de desempenho (SRS §20.7) para as tabelas desta tarefa.
CREATE INDEX idx_auditoria_quando  ON auditoria (criado_em DESC);
CREATE INDEX idx_auditoria_usuario ON auditoria (usuario_id, criado_em DESC);
CREATE INDEX idx_notif_abertas     ON notificacao (usuario_id) WHERE lida_em IS NULL;

-- Semente fixa de perfis (códigos da SRS §9). Não é credencial (RS-004):
-- só códigos/nomes; nenhum usuário ou senha.
INSERT INTO perfil (codigo, nome) VALUES
  ('SEC', 'Secretaria'),
  ('FIN', 'Financeiro'),
  ('COO', 'Coordenação'),
  ('CPR', 'Compras/Almoxarifado'),
  ('RH',  'Recursos Humanos'),
  ('DIR', 'Direção'),
  ('ADM', 'Administrador'),
  ('PRO', 'Professor'),
  ('RES', 'Responsável')
ON CONFLICT (codigo) DO NOTHING;

-- Privilégio mínimo (RS-024): aplicação acessa o esquema e tem só o DML
-- necessário nas tabelas existentes (inclui a tabela de versão do goose,
-- que a aplicação nunca usa).
GRANT USAGE ON SCHEMA public TO sacre_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO sacre_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sacre_app;

-- Auditoria append-only (RS-011): o papel de aplicação só insere.
REVOKE UPDATE, DELETE, TRUNCATE ON auditoria FROM sacre_app;

-- Tabelas futuras (próximas migrações, mesmo dono) herdam o mesmo mínimo.
ALTER DEFAULT PRIVILEGES FOR ROLE sacre_migracao IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE ON TABLES TO sacre_app;
ALTER DEFAULT PRIVILEGES FOR ROLE sacre_migracao IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO sacre_app;

-- +goose Down

-- Rollback da fundação: remove as tabelas na ordem reversa das FKs.
-- Papéis e extensão são mantidos de propósito (podem ser donos de outros
-- objetos; recriá-los é idempotente no Up).
DROP TABLE IF EXISTS notificacao;
DROP TABLE IF EXISTS auditoria;
DROP TABLE IF EXISTS usuario_perfil;
DROP TABLE IF EXISTS perfil;
DROP TABLE IF EXISTS usuario;
