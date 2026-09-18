-- 0003_conta.sql — invalidação de sessões por carimbo (F-05).
--
-- RS-006 (troca/redefinição encerra todas as sessões) via
-- `usuario.sessoes_invalidas_antes_de`: a sessão carrega `criada_em` (scs) e
-- o middleware/handlers a comparam com este carimbo — sessão mais antiga que
-- o carimbo é tratada como inexistente (401). Expansão pura (coluna anulável,
-- sem default): compatível com a versão anterior (AGENTS.md §7).
-- Nenhuma credencial (RS-004).
--
-- Nota: `sessao_usuario` (F-04) mapeia token→usuário, mas o vínculo no
-- handler viola a FK (o commit do scs só acontece ao final do LoadAndSave);
-- o carimbo é o mecanismo real de RS-006. A tabela segue existindo sem
-- escrita nova; remoção fica para migração de contração futura.

-- +goose Up

ALTER TABLE usuario ADD COLUMN sessoes_invalidas_antes_de timestamptz NULL;

-- +goose Down

ALTER TABLE usuario DROP COLUMN sessoes_invalidas_antes_de;
