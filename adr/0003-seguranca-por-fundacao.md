# ADR-0003 — Segurança por fundação: sessão, TOTP, RBAC e auditoria com bibliotecas prontas

- **Status:** Aceito (stack definida pelo autor do projeto; não reaberta)
- **Data:** 16/09/2026
- **Relacionados:** PRD §4 (Épico 0), SRS §4 (requisitos de segurança), THREAT-MODEL §4, ADR-0001 (backend)

## Contexto

O sistema guarda **dados pessoais de alunos menores de idade e de seus responsáveis** (LGPD, art. 14 — dados de crianças e adolescentes). A segurança é requisito de arquitetura (Diretiva 1), não etapa final: cada ameaça do THREAT-MODEL precisa gerar um requisito rastreável, e o código será escrito por IA — tratado como não confiável até passar pelas verificações. Construir criptografia, hash de senha ou autenticação do zero é proibido.

## Decisão

- **Hash de senha:** `alexedwards/argon2id` (nunca implementação própria).
- **Sessões:** `alexedwards/scs v2` com store PostgreSQL (`pgxstore`); cookie `__Host-` com `HttpOnly`, `Secure`, `SameSite=Strict`; tempo máximo de inatividade e tempo absoluto; renovação do identificador no login e na troca de privilégio; encerramento de todas as sessões do usuário na troca de senha.
- **2FA (TOTP):** `pquerna/otp`; obrigatório para Direção, Financeiro, RH e administrador; opcional para os demais. O segredo TOTP é cifrado com **`golang.org/x/crypto/nacl/secretbox`**, com chave em variável de ambiente exclusiva do ambiente Production. Procedimento de recuperação para perda do autenticador definido no SRS (revalidação por administrador + auditoria).
- **CSRF:** `http.CrossOriginProtection` da biblioteca padrão (motivo do piso Go 1.25).
- **Autorização:** RBAC com permissões declaradas como *scopes* no OpenAPI; perfis e permissões no banco; verificação num único middleware; negar por padrão; autorização sempre no backend (guards do Vue Router são só UX). Não usamos provedor de identidade externo (decisão registrada aqui como ADR): um IdP externo gratuito acrescentaria dependência de disponibilidade de terceiro, complexidade de integração para a IA e perda do controle do cookie first-party — sem ganho de segurança proporcional para um MVP com perfis internos bem definidos.
- **Rate limiting:** `go-chi/httprate` com contador no PostgreSQL (serverless não compartilha memória entre instâncias); no login, bloqueio progressivo por conta. Regras do firewall da Vercel como camada extra, se o plano permitir `[VERIFICAR]`.
- **Cabeçalhos de segurança:** `unrolled/secure` nas respostas da API; bloco `headers` do `vercel.json` para os estáticos.
- **Validação de documentos:** CPF/CNPJ com `paemuri/brdoc` no backend.
- **Auditoria:** tabela append-only (só inserção; papel de aplicação não pode alterar nem apagar); registra login, falhas de login, alteração de nota, baixa de pagamento, mudança de permissão, redefinição de senha e exportação de dados; sem senhas, tokens ou códigos TOTP nos logs.
- **Primeiro administrador:** criado por comando administrativo executado pelo GitHub Actions, senha temporária vinda de secret, troca obrigatória no primeiro login; nunca há usuário/senha padrão em código, seeds de produção ou migrações.
- **Redefinição de senha no MVP:** por administrador (não há e-mail), senha temporária de uso único, troca no próximo login, registro na auditoria.

## Alternativas consideradas

| Alternativa | Por que não |
| :--- | :--- |
| Implementar argon2/TOTP/secretbox do zero | Proibido pela Diretiva 1; erro em criptografia caseira é catastrófico e difícil de detectar por revisão |
| JWT + localStorage | Persiste token no navegador (proibido); revogação difícil; cookie de sessão first-party é mais simples e seguro aqui |
| IdP externo (Auth0, Cognito, Keycloak) | Custo/complexidade ou self-host fora do plano; perde o cookie first-party; registrado como ADR conforme Diretiva 1 |
| Rate limit em memória | Serverless: contadores não compartilham entre instâncias; contador no PostgreSQL é compartilhado |
| Auditoria com alteração (upsert) | Violaria imutabilidade; append-only é verificável no banco via permissões do papel de aplicação |

## Consequências

- **Positivas:** bibliotecas maduras e conhecidas pelos modelos (Diretiva 2); requisitos verificáveis por teste (cada perfil tentando acessar o que não pode); auditoria resistente ao próprio papel de aplicação.
- **Negativas:** TOTP obrigatório a quatro perfis adiciona atrito no primeiro uso (mitigado pelo fluxo de cadastro do autenticador guiado); chave `secretbox` exclusiva de Production cria dependência de gestão de segredos bem feita.
- **Neutras:** a chave `__Host-` exige HTTPS em qualquer ambiente (previews e produção atendem por HTTPS por padrão).
