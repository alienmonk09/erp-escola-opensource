-- testdata/fixtures_fundacao.sql — fixtures fictícias da fundação (F-03).
--
-- RE-06 / AGENTS.md §4.10: dados 100% fictícios e versionados em git; nenhum
-- dado real de aluno/responsável. Domínio `.test` (RFC 2606, nunca real).
-- RS-004: `hash_senha` é um marcador NÃO utilizável (não é hash válido nem
-- credencial); autenticação real só chega em F-04/F-05.
-- Estes UUIDs fixos existem só para tornar os testes determinísticos.

INSERT INTO usuario (id, nome, email, hash_senha, senha_temporaria, troca_obrigatoria, totp_secreto_cifrado, totp_ativo, ativo)
VALUES (
  '11111111-1111-4111-8111-111111111111',
  'Helena Fictícia da Silva',
  'helena.ficticia@exemplo-escola.test',
  'ARGON2ID_FICTICIO_NAO_UTILIZAVEL_EM_TESTE',
  false,
  false,
  NULL,
  false,
  true
);

INSERT INTO usuario_perfil (usuario_id, perfil_id)
VALUES (
  '11111111-1111-4111-8111-111111111111',
  (SELECT id FROM perfil WHERE codigo = 'SEC')
);

INSERT INTO notificacao (usuario_id, titulo, mensagem)
VALUES (
  '11111111-1111-4111-8111-111111111111',
  'Boas-vindas (dado fictício)',
  'Notificação de exemplo para o teste de integração da F-03.'
);
