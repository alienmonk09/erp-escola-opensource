-- queries/auth.sql — sessão, bloqueio e rate limit (F-04). RS-031.

-- name: ListarCodigosPerfilDoUsuario :many
SELECT p.codigo
FROM usuario_perfil up
JOIN perfil p ON p.id = up.perfil_id
WHERE up.usuario_id = $1;

-- name: GetLoginBloqueio :one
SELECT email, falhas, bloqueado_ate, atualizado_em
FROM login_bloqueio
WHERE email = $1;

-- name: UpsertLoginFalha :one
INSERT INTO login_bloqueio (email, falhas, bloqueado_ate, atualizado_em)
VALUES ($1, 1, $2, now())
ON CONFLICT (email) DO UPDATE SET
  falhas = login_bloqueio.falhas + 1,
  bloqueado_ate = EXCLUDED.bloqueado_ate,
  atualizado_em = now()
RETURNING email, falhas, bloqueado_ate, atualizado_em;

-- name: ResetLoginBloqueio :exec
DELETE FROM login_bloqueio WHERE email = $1;

-- name: VincularSessaoUsuario :exec
INSERT INTO sessao_usuario (token, usuario_id)
VALUES ($1, $2)
ON CONFLICT (token) DO UPDATE SET usuario_id = EXCLUDED.usuario_id;

-- name: EncerrarSessoesDoUsuario :exec
DELETE FROM sessions
WHERE token IN (
  SELECT token FROM sessao_usuario WHERE usuario_id = $1
);

-- name: IncrementarHttprate :one
INSERT INTO httprate_contador (chave, janela, contagem)
VALUES ($1, $2, 1)
ON CONFLICT (chave, janela) DO UPDATE SET
  contagem = httprate_contador.contagem + 1
RETURNING contagem;

-- name: GetHttprate :one
SELECT contagem
FROM httprate_contador
WHERE chave = $1 AND janela = $2;
