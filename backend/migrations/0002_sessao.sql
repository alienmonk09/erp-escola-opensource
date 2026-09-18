-- 0002_sessao.sql — sessão scs, rate limit e bloqueio progressivo (F-04).
--
-- Tabelas de suporte a RS-003 (scs/pgxstore), RS-012/RS-050/RS-051
-- (contador httprate + bloqueio por conta) e RS-006 (encerrar todas as
-- sessões do usuário). Nenhuma credencial (RS-004).

-- +goose Up

CREATE TABLE sessions (
  token  text        PRIMARY KEY,
  data   bytea       NOT NULL,
  expiry timestamptz NOT NULL
);
CREATE INDEX idx_sessions_expiry ON sessions (expiry);

CREATE TABLE sessao_usuario (
  token      text        PRIMARY KEY REFERENCES sessions (token) ON DELETE CASCADE,
  usuario_id uuid        NOT NULL REFERENCES usuario (id) ON DELETE RESTRICT,
  criado_em  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessao_usuario_usuario ON sessao_usuario (usuario_id);

CREATE TABLE login_bloqueio (
  email          text        PRIMARY KEY,
  falhas         integer     NOT NULL DEFAULT 0,
  bloqueado_ate  timestamptz NULL,
  atualizado_em  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE httprate_contador (
  chave          text        NOT NULL,
  janela         timestamptz NOT NULL,
  contagem       bigint      NOT NULL DEFAULT 0,
  PRIMARY KEY (chave, janela)
);

GRANT SELECT, INSERT, UPDATE, DELETE ON sessions TO sacre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sessao_usuario TO sacre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON login_bloqueio TO sacre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON httprate_contador TO sacre_app;

-- +goose Down

DROP TABLE IF EXISTS httprate_contador;
DROP TABLE IF EXISTS login_bloqueio;
DROP TABLE IF EXISTS sessao_usuario;
DROP TABLE IF EXISTS sessions;
