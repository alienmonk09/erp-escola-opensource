-- queries/fundacao.sql — queries mínimas da fundação (F-03).
--
-- Requisitos: RS-031 (só SQL parametrizado via sqlc/pgx; sem concatenação).
-- Escopo desta tarefa: buscar usuário por e-mail (base do login em F-04) e
-- inserir registro de auditoria (RS-010). Novas queries entram aqui tarefa a
-- tarefa e o código é regenerado com `task gerar:sql` (AGENTS.md §6).

-- name: GetUsuarioPorEmail :one
SELECT id, nome, email, hash_senha, senha_temporaria, troca_obrigatoria,
       totp_secreto_cifrado, totp_ativo, ativo, criado_em, atualizado_em
FROM usuario
WHERE email = $1
LIMIT 1;

-- name: InsertAuditoria :one
INSERT INTO auditoria (usuario_id, acao, recurso, detalhe, ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, usuario_id, acao, recurso, detalhe, ip, criado_em;
