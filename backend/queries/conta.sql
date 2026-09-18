-- queries/conta.sql — conta própria e gestão de usuários (F-05). RS-031.
--
-- Requisitos: RF-005, RF-007; RS-006..009 (troca encerra sessões, temporária
-- de uso único com troca obrigatória, TOTP com segredo cifrado).
-- Só SQL parametrizado via sqlc/pgx; sem concatenação.

-- name: GetUsuarioPorID :one
SELECT id, nome, email, hash_senha, senha_temporaria, troca_obrigatoria,
       totp_secreto_cifrado, totp_ativo, ativo, criado_em, atualizado_em,
       sessoes_invalidas_antes_de
FROM usuario
WHERE id = $1
LIMIT 1;

-- name: TrocarSenhaPropria :exec
UPDATE usuario
SET hash_senha = $2,
    senha_temporaria = false,
    troca_obrigatoria = false,
    atualizado_em = now()
WHERE id = $1;

-- name: RedefinirSenhaAdmin :exec
UPDATE usuario
SET hash_senha = $2,
    senha_temporaria = true,
    troca_obrigatoria = true,
    atualizado_em = now()
WHERE id = $1;

-- name: DefinirTotpSecreto :exec
UPDATE usuario
SET totp_secreto_cifrado = $2,
    atualizado_em = now()
WHERE id = $1;

-- name: AtivarTotp :exec
UPDATE usuario
SET totp_ativo = true,
    atualizado_em = now()
WHERE id = $1;

-- name: ReiniciarTotp :exec
UPDATE usuario
SET totp_secreto_cifrado = NULL,
    totp_ativo = false,
    atualizado_em = now()
WHERE id = $1;

-- name: InvalidarSessoesDoUsuario :exec
UPDATE usuario
SET sessoes_invalidas_antes_de = $2,
    atualizado_em = now()
WHERE id = $1;
