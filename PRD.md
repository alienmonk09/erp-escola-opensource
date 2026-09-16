# PRD — ERP Escola Sacre Coeur Des Enfants

| | |
| :--- | :--- |
| **Produto** | ERP escolar de código aberto para a Escola Sacre Cœur des Enfants (São Luís, MA) |
| **Versão do documento** | 1.0 — Etapa 1 |
| **Data** | 16/09/2026 |
| **Autoria** | Requisitos, arquitetura e modelagem de ameaças conduzidos por IA (Freebuff), sob especificação e revisão de Jader Augusto Maciel Fonseca |
| **Documentos relacionados** | `SRS.md`, `THREAT-MODEL.md`, `ROADMAP.md`, `TUTORIAL.md`, `adr/` |
| **Fonte primária** | `docs/referencias/entrega-erp-sacre-coeur.md` e `docs/referencias/fluxograma-erp-sacre-coeur.png` |

> **Como usar este documento.** Todos os requisitos e ameaças têm IDs rastreáveis: US-XXX (história), RF-XXX (funcional), RNF-XXX (não funcional), RS-XXX (segurança), RP-XXX (privacidade/LGPD), RN-XXX (regra de negócio), AM-XXX (ameaça). A matriz completa de rastreabilidade está na seção 9.

---

## 1. Visão do produto

Dar à Escola Sacre Cœur des Enfants um sistema de gestão integrada (ERP) em que **todos os setores compartilham um único banco de dados**: o aluno, o responsável, o professor e o material são cadastrados uma única vez, e cada alteração feita em um módulo fica imediatamente visível nos demais, sem retrabalho, sem planilhas paralelas e sem depender de alguém "avisar" o outro setor.

O produto existe para **provar duas teses**:

1. **Viabilidade de custo zero:** um ERP completo, do zero até o deploy, construído com ferramentas, modelos de IA e hospedagem 100% gratuitas e/ou open source (Diretiva 3).
2. **Viabilidade do desenvolvimento sem escrita humana de código:** a IA conduz o fluxo inteiro de criação do software; o papel humano é especificar, revisar e aprovar (Diretiva 4).

A stack tecnológica já está definida pelo autor do projeto (seção "Stack definida" do prompt) e **não será reaberta** em nenhum documento desta etapa. As justificativas estão nos ADRs de `adr/0001` a `adr/0004`.

### 1.1 Problema

A escola cresceu, mas a forma de organizar as informações continuou a mesma: cada setor controla seus próprios dados (planilhas, sistemas isolados), sem integração. Consequências registradas no diagnóstico:

- **Informações duplicadas:** o mesmo aluno é digitado na planilha da secretaria, no sistema do financeiro e no controle da coordenação.
- **Dificuldade de achar o dado atualizado:** com três cadastros do mesmo aluno, ninguém sabe qual está certo.
- **Demora na comunicação:** o responsável paga a mensalidade, mas a secretaria continua vendo o aluno como pendente até alguém do financeiro avisar.
- **Retrabalho:** a mesma matrícula precisa ser digitada de novo no financeiro e na coordenação.
- **Compras sem ver o estoque:** o setor de compras pede material que já existe no almoxarifado, ou deixa faltar.

**Causa principal:** a escola não tem uma base de dados compartilhada — um banco de dados para cada área, informação descentralizada, processos isolados.

### 1.2 Objetivos e métricas de sucesso

| # | Objetivo | Métrica de sucesso | Meta |
| :--- | :--- | :--- | :--- |
| OBJ-1 | Cadastro único e compartilhado | Nº de cadastros duplicados do mesmo aluno no sistema | **0** (integridade garantida por chave única no banco) |
| OBJ-2 | Comunicação automática entre setores | Tempo entre a baixa do pagamento e a atualização da situação do aluno para a Secretaria | ≤ tempo de invalidação de cache (RNF-013: ≤ 60 s sem ação do usuário) |
| OBJ-3 | Fim do retrabalho de digitação | Passos manuais para efetivar uma matrícula | 1 único cadastro gera contrato, mensalidades e vaga (RN-001) |
| OBJ-4 | Compras guiadas pelo estoque real | Requisições que geram compra apenas com saldo insuficiente | 100% (RN-006) |
| OBJ-5 | Visão gerencial consolidada | Indicadores da Direção atualizados | Inadimplência, alunos por turma, gastos e estoque em um só painel (RF-601..605) |
| OBJ-6 | Custo zero comprovado | Custo mensal em dinheiro de infraestrutura, agentes e hospedagem | **R$ 0,00** (apenas planos gratuitos permanentes e open source) |
| OBJ-7 | Desenvolvimento sem código humano | Linhas de código de produção escritas manualmente por humanos | **0** (humanos escrevem apenas especificação, revisão e aprovação) |
| OBJ-8 | Segurança por fundação | PRs mesclados com falha de segurança ignorando gate do CI | **0** (gates bloqueiam merge — RS-030..034) |

## 2. Personas

| Persona | Setor | O que faz hoje | O que precisa do ERP |
| :--- | :--- | :--- | :--- |
| **Maria — Secretaria** | Secretaria | Digita matrículas em planilha; reaproveita dados de outras planilhas | Cadastrar aluno/responsável **uma vez**; efetivar matrícula que gere contrato e mensalidades automaticamente; ver a situação financeira do aluno sem esperar aviso do Financeiro |
| **Carlos — Financeiro** | Financeiro | Sistema próprio de mensalidades; avisa a Secretaria por fora quando um pagamento cai | Registrar baixas de pagamento; ver inadimplência; pagar fornecedores e folha; emitir boleto demonstrativo |
| **Ana — Coordenação Pedagógica** | Coordenação | Controle acadêmico separado; monta turmas com base em lista da Secretaria | Montar turmas a partir dos alunos matriculados; lançar notas e frequência; gerar boletim e histórico |
| **João — Compras/Almoxarifado** | Compras e Estoque | Controle próprio; compra sem enxergar o saldo real | Registrar requisições de material que consultem o saldo; aprovar e receber pedidos; manter o estoque atualizado |
| **Paula — RH** | RH | Cadastro de professores/funcionários e folha em controles próprios | Cadastrar professores e funcionários, contratos e carga horária; processar folha; alocar professores às turmas |
| **Sandra — Direção** | Direção | Recebe informação consolidada manualmente, tarde e às vezes errada | Consultar painéis e indicadores de todos os módulos e **exportar relatórios** (com auditoria) |
| **Responsável/Aluno** | — | Recebe informação por contato direto | Consulta **apenas leitura** da própria situação: situação financeira, boletim e avisos internos da escola |

> **Assunção A-01:** a existência do portal do responsável/aluno como consulta somente leitura é uma suposição registrada, pois não está explícita no documento-fonte. Ela aparece no fluxograma apenas implicitamente. Fica no MVP como leitura pura, sem interação (sem "recado", sem pagamento online).

> **Assunção A-02:** os funcionários dos setores têm acesso a computador com navegador moderno e internet na escola. Não há requisito de aplicativo móvel nativo; a interface é responsiva para uso eventual em celular (RNF-011).

> **Assunção A-03:** o tamanho da escola (nº de alunos, turmas, funcionários) não está especificado no documento-fonte. As metas de desempenho e as estimativas de cota (SRS, RNF) assumem uma escola de pequeno a médio porte (ordem de centenas de alunos). Se a escola for maior, as metas devem ser revisadas na Etapa 2.

## 3. Escopo do MVP

### 3.1 Dentro do MVP

Os seis módulos do fluxograma, sobre um **banco de dados único**, com os fluxos entre módulos listados na seção 2.2 do documento-fonte:

| Módulo | Escopo do MVP |
| :--- | :--- |
| **Acadêmico** (Secretaria) | Cadastro único de alunos e responsáveis, matrícula, rematrícula, documentos e histórico escolar |
| **Pedagógico** (Coordenação) | Turmas, professores por turma, notas, frequência e boletim |
| **Financeiro** (Financeiro) | Mensalidades, boletos **demonstrativos** (sem registro bancário), baixa de pagamentos, inadimplência, pagamento de fornecedores e da folha |
| **Compras e Estoque** (Compras/Almoxarifado) | Pedidos de material, fornecedores, entrada e saída de material, saldo do estoque e aviso de reposição |
| **RH** | Cadastro de professores e funcionários, contratos, carga horária e folha de pagamento |
| **Gerencial** (Direção) | Relatórios e painéis com dados de todos os módulos |
| **Transversal** | Login com senha + TOTP (obrigatório p/ Direção, Financeiro, RH e administrador; opcional p/ demais), RBAC, auditoria, notificações internas (tabela), dashboard inicial |

### 3.2 Fora do MVP (com justificativa)

| Item | Justificativa |
| :--- | :--- |
| **Integração bancária real (boleto registrável, Pix, conciliação)** | Exige contrato com banco, credenciais e domínio próprio; o boleto emitido é **demonstrativo**. A baixa de pagamento é manual pelo Financeiro |
| **E-mail / notificações externas** | Exige domínio próprio e serviço externo; no MVP, notificações são **apenas internas** (tabela). Por isso a redefinição de senha é feita por administrador |
| **Pagamento online pelo responsável** | Depende de integração de pagamento (custo, PCI, LGPD); fora do MVP |
| **Diário de classe, conteúdo programático, biblioteca, ocorrências disciplinares** | Não aparecem nos fluxos do documento-fonte nem no fluxograma |
| **Contratação/admissão formal (eSocial, benefícios, ponto)** | Fora do fluxo do documento; RH cobre cadastro, contratos, carga horária e folha **demonstrativa** (sem integração governamental) |
| **Aplicativo móvel nativo / PWA instalável** | Interface responsiva basta; não há requisito no documento-fonte |
| **Integração com governos (SEESP, SEBRAE, prefeitura), SIS/SEB** | Fora do escopo acadêmico do projeto |
| **Múltiplas escolas / multi-tenant** | O sistema é para **uma** escola (Sacre Cœur) |
| **Chat/mensageria entre setores** | A comunicação entre setores acontece pela **troca de dados**, não por mensagens |

## 4. Épicos e histórias de usuário

Formato: "Como… quero… para…", com critérios de aceite em Given/When/Then, incluindo critérios de segurança quando aplicável. Priorização MoSCoW na seção 5.

### Épico 0 — Fundação e acesso seguro (transversal)

- **US-001 (Must)** — Como **qualquer usuário do sistema**, quero **fazer login com minha senha e, quando exigido, meu segundo fator (TOTP)**, para **acessar somente o que o meu perfil permite**.
  - **Given** usuário com conta ativa e senha correta, **When** submete o login, **Then** o sistema pede o código TOTP se o perfil exigir e cria a sessão.
  - **Given** senha errada, **When** submete o login, **Then** recebe mensagem genérica de credenciais inválidas (sem revelar se o usuário existe) e a tentativa é registrada na auditoria (RS-001, RS-013).
  - **Given** perfil Direção, Financeiro, RH ou administrador, **When** faz login **sem** registrar o segundo fator, **Then** o acesso é bloqueado até concluir o cadastro do TOTP.
  - **Given** perfil com TOTP opcional (Secretaria, Coordenação, Compras), **When** opta por habilitar, **Then** o sistema segue a mesma rotina de segundo fator.
  - **Given** usuário tenta fazer login sem segundo fator em perfil que exige, **When** submete só a senha, **Then** recebe instrução para configurar o TOTP e não recebe token de sessão.

- **US-002 (Must)** — Como **usuário logado**, quero **ter minha sessão encerrada após tempo de inatividade e após tempo absoluto**, para **reduzir o risco de uso indevido da minha conta**.
  - **Given** sessão ativa, **When** passa do tempo máximo de inatividade, **Then** a próxima requisição exige novo login.
  - **Given** sessão ativa, **When** passa do tempo máximo absoluto, **Then** a sessão é encerrada mesmo com atividade contínua.
  - **Given** usuário troca de senha, **When** a troca é confirmada, **Then** **todas** as sessões daquele usuário são encerradas (RS-006).
  - **Given** usuário faz login, **When** a sessão é criada, **Then** o identificador da sessão é renovado em relação a qualquer sessão anterior (RS-005).

- **US-003 (Must)** — Como **administrador**, quero **criar o primeiro administrador por um comando administrativo** (executado pelo GitHub Actions, com senha temporária vinda de secret), para **nunca existir usuário ou senha padrão no código, nos seeds de produção ou nas migrações**.
  - **Given** sistema recém-implantado sem administrador, **When** o comando é executado com a senha temporária do secret, **Then** o administrador é criado com obrigação de trocar a senha no primeiro login.
  - **Given** sistema já tem administrador, **When** o comando é executado novamente, **Then** ele se recusa a criar outro (idempotente).
  - **Given** qualquer deploy, **When** é executado, **Then** não há nenhum usuário ou credencial padrão criado por código, seed ou migração (RS-004).

- **US-004 (Must)** — Como **administrador**, quero **redefinir a senha de um usuário gerando uma senha temporária de uso único**, para **recuperar o acesso de quem perdeu a senha sem e-mail no MVP**.
  - **Given** usuário esqueceu a senha, **When** o administrador solicita a redefinição, **Then** o sistema gera senha temporária de uso único, exibe ao administrador, e força a troca no próximo login do usuário.
  - **Given** redefinição concluída, **When** a senha é trocada, **Then** todas as sessões do usuário são encerradas e a ação fica registrada na auditoria (RS-007, RS-010).

- **US-005 (Must)** — Como **usuário com perfil que exige TOTP**, quero **ter um procedimento de recuperação quando perder o autenticador**, para **não perder acesso definitivamente**.
  - **Given** usuário com TOTP obrigatório perdeu o autenticador, **When** solicita recuperação, **Then** um administrador revalida a identidade e reinicia o cadastro do TOTP; a ação fica registrada na auditoria (RS-008).

### Épico 1 — Acadêmico (Secretaria)

- **US-101 (Must)** — Como **secretaria**, quero **cadastrar o aluno e seus responsáveis uma única vez**, para **que todos os módulos usem o mesmo registro**.
  - **Given** aluno novo, **When** a secretaria cadastra com dados pessoais e responsáveis, **Then** o registro é criado uma única vez e fica visível nos outros módulos.
  - **Given** tentativa de cadastro de aluno com CPF já existente, **When** submete o formulário, **Then** o sistema recusa com mensagem clara (RN-002).
  - **Given** validação de CPF inválido, **When** submete, **Then** o sistema recusa antes de gravar (paemuri/brdoc no backend).
  - **Segurança:** endpoint exige autenticação e scope `academico:alunos:escrever`; dados pessoais de menor de idade recebem tratamento LGPD (RP-002, RP-003).

- **US-102 (Must)** — Como **secretaria**, quero **efetivar a matrícula do aluno em uma série/ano letivo**, para **gerar automaticamente o contrato e as mensalidades no Financeiro**.
  - **Given** aluno cadastrado, **When** a secretaria efetiva a matrícula, **Then** o sistema cria o contrato e as mensalidades no Financeiro e a vaga na série (RN-001).
  - **Given** aluno com matrícula ativa naquele ano letivo, **When** tenta matricular de novo, **Then** o sistema recusa (RN-003).
  - **Segurança:** exige scope `academico:matriculas:escrever`; auditoria registra quem efetivou (RS-010).

- **US-103 (Must)** — Como **secretaria**, quero **ver a situação financeira do aluno sem esperar aviso do Financeiro**, para **informar corretamente o responsável**.
  - **Given** Financeiro deu baixa em um pagamento, **When** a secretaria consulta o aluno, **Then** a situação aparece atualizada em ≤ 60 s sem ação manual (OBJ-2, RNF-013).
  - **Given** aluno inadimplente, **When** a secretaria consulta, **Then** a situação "pendente/inadimplente" fica visível.
  - **Segurança:** leitura de situação financeira exige scope `academico:alunos:ler` + permissão explícita para ver dados financeiros de alunos (privilégio mínimo).

- **US-104 (Should)** — Como **secretaria**, quero **registrar e consultar documentos do aluno (matrícula, histórico escolar)**, para **cumprir as exigências escolares**.
  - **Given** documento anexado ou registrado, **When** consultado, **Then** aparece no cadastro do aluno com histórico de alterações (auditoria RS-010).
  - **Segurança:** dados de documento de menor de idade = dado pessoal sensível para fins de acesso; acesso por scope específico.

- **US-105 (Should)** — Como **secretaria**, quero **efetuar a rematrícula do aluno para o ano letivo seguinte**, para **renovar contrato e mensalidades sem redigitar dados**.
  - **Given** aluno matriculado no ano atual, **When** rematrículado, **Then** gera novo contrato e mensalidades para o ano seguinte (RN-001) sem redigitação.

### Épico 2 — Pedagógico (Coordenação)

- **US-201 (Must)** — Como **coordenação**, quero **montar turmas a partir dos alunos matriculados por série**, para **distribuir as vagas sem planilha paralela**.
  - **Given** alunos matriculados em uma série, **When** a coordenação monta a turma, **Then** a turma é criada com os alunos daquela série e professores alocados.
  - **Given** aluno sem matrícula ativa, **When** tenta incluí-lo em turma, **Then** o sistema recusa (RN-004).
  - **Segurança:** scope `pedagogico:turmas:escrever`.

- **US-202 (Must)** — Como **coordenação**, quero **lançar notas e frequência dos alunos**, para **acompanhar o desempenho e alimentar o boletim**.
  - **Given** professor/coordenador lança nota, **When** confirmada, **Then** fica registrada com autor e data na auditoria (RS-010, AM-005).
  - **Given** nota já lançada, **When** alterada, **Then** a alteração exige permissão específica e fica registrada (quem alterou, quando, valor anterior — sem conter dado pessoal desnecessário no log).
  - **Segurança:** scope `pedagogico:notas:escrever`; professor só lança nota para as turmas em que leciona (privilégio mínimo — RN-005).

- **US-203 (Must)** — Como **coordenação ou professor**, quero **gerar o boletim do aluno em PDF**, para **entregar ao responsável**.
  - **Given** aluno com notas lançadas, **When** o boletim é gerado, **Then** o PDF é produzido na requisição (maroto v2) e devolvido direto na resposta, sem gravar em disco.
  - **Segurança:** scope `pedagogico:boletim:ler`; responsável acessa **somente** o boletim do próprio filho (RP-006).

- **US-204 (Should)** — Como **coordenação**, quero **ver quantos professores cada turma precisa e alocar os contratados pelo RH**, para **fechar a grade**.
  - **Given** turmas criadas, **When** a coordenação consulta, **Then** o sistema mostra professores por turma e a carga horária alocada (fluxo Coordenação ↔ RH).
  - **Segurança:** leitura de professores e carga horária exige scope `rh:professores:ler` (dado pessoal de funcionário — RP-004).

### Épico 3 — Financeiro (Financeiro)

- **US-301 (Must)** — Como **financeiro**, quero **ter as mensalidades geradas automaticamente quando a matrícula é efetivada**, para **não digitá-las de novo**.
  - **Given** matrícula efetivada, **When** o sistema gera as mensalidades, **Then** cada mensalidade aparece com vencimento e valor em **centavos (inteiro, BRL)** — nunca ponto flutuante (RNF-005).
  - **Given** matrícula cancelada antes do início das aulas, **When** ocorre, **Then** as mensalidades geradas podem ser canceladas pelo Financeiro (RN-001 detalhada no SRS).

- **US-302 (Must)** — Como **financeiro**, quero **registrar a baixa de um pagamento**, para **atualizar a situação do aluno na hora**.
  - **Given** mensalidade pendente, **When** o Financeiro registra a baixa, **Then** a situação do aluno muda para "em dia" no mesmo instante para todos os módulos (OBJ-2).
  - **Given** baixa registrada, **When** confirmada, **Then** auditoria registra quem deu a baixa, quando e qual título (RS-010, AM-006).
  - **Segurança:** scope `financeiro:baixas:escrever`; auditoria append-only (RS-011).

- **US-303 (Must)** — Como **financeiro**, quero **emitir boleto demonstrativo em PDF**, para **entregar ao responsável para pagamento externo**.
  - **Given** mensalidade pendente, **When** o Financeiro emite o boleto, **Then** o PDF é gerado na requisição (maroto v2), **sem registro bancário** e sem integração com banco.
  - **Segurança:** scope `financeiro:boletos:ler`; boleto contém dados pessoais do responsável — acesso mínimo (RP-003).

- **US-304 (Must)** — Como **financeiro**, quero **consultar a inadimplência por aluno, turma e escola**, para **cobrar e reportar à Direção**.
  - **Given** alunos com mensalidades vencidas não pagas, **When** o Financeiro consulta, **Then** a inadimplência aparece por aluno, turma e total (RFC para o Gerencial).

- **US-305 (Must)** — Como **financeiro**, quero **registrar o pagamento de fornecedores e da folha**, para **manter o fluxo de caixa atualizado**.
  - **Given** pedido aprovado e nota fiscal lançada, **When** o Financeiro paga o fornecedor, **Then** a baixa aparece no fluxo de caixa (fluxo Compras → Financeiro).
  - **Given** folha processada pelo RH, **When** o Financeiro paga, **Then** a baixa é registrada (fluxo RH → Financeiro).
  - **Segurança:** scope `financeiro:pagamentos:escrever`; auditoria.

### Épico 4 — Compras e Estoque

- **US-401 (Must)** — Como **qualquer setor**, quero **abrir uma requisição de material**, para **que a Compras saiba o que foi pedido**.
  - **Given** setor precisa de material, **When** abre a requisição, **Then** o sistema registra item, quantidade e justificativa (fluxo Coordenação/setores → Compras).
  - **Segurança:** scope `compras:requisicoes:escrever`.

- **US-402 (Must)** — Como **compras**, quero **que a requisição consulte o saldo do estoque antes de gerar compra**, para **não comprar o que já existe nem deixar faltar**.
  - **Given** requisição com item com saldo suficiente, **When** o Compras avalia, **Then** o sistema indica que pode atender do estoque e **não** gera compra (RN-006).
  - **Given** requisição com item com saldo insuficiente, **When** avaliada, **Then** o sistema sugere a quantidade a comprar.
  - **Given** estoque abaixo do mínimo, **When** qualquer saída ocorre, **Then** o sistema emite aviso de reposição (OBJ-4).

- **US-403 (Must)** — Como **compras**, quero **cadastrar fornecedores e aprovar pedidos de compra**, para **formalizar a compra**.
  - **Given** pedido aprovado com nota fiscal, **When** enviado ao Financeiro, **Then** o Financeiro vê o título a pagar (fluxo Compras → Financeiro).
  - **Segurança:** scope `compras:pedidos:aprov`.

- **US-404 (Must)** — Como **almoxarifado**, quero **registrar a entrada e a saída de material**, para **manter o saldo real do estoque**.
  - **Given** material recebido, **When** registrado, **Then** o saldo do estoque aumenta (fluxo Compras → Almoxarifado).
  - **Given** material entregue a um setor, **When** registrado, **Then** o saldo diminui e o aviso de reposição é avaliado (RN-006).
  - **Segurança:** scope `almoxarifado:movimentos:escrever`; auditoria em movimentos.

### Épico 5 — RH

- **US-501 (Must)** — Como **RH**, quero **cadastrar professores e funcionários com contrato e carga horária**, para **distribuir as aulas e pagar a folha**.
  - **Given** professor contratado, **When** cadastrado, **Then** o cadastro aparece para a Coordenação alocar (fluxo RH → Coordenação).
  - **Segurança:** dados pessoais de funcionários — escopo `rh:funcionarios:escrever`; dados sensíveis minimizados (RP-004).

- **US-502 (Must)** — Como **RH**, quero **processar a folha de pagamento e enviá-la ao Financeiro**, para **que os salários sejam pagos**.
  - **Given** contratos e carga horária lançados, **When** a folha é processada, **Then** o Financeiro recebe a folha a pagar (fluxo RH → Financeiro).
  - **Segurança:** scope `rh:folha:escrever`; dados remuneratórios = dado pessoal sensível para fins de acesso (RP-004).

- **US-503 (Should)** — Como **coordenação**, quero **ver quantos professores cada turma precisa**, para **solicitar contratação ou realocação ao RH**.
  - **Given** turmas criadas, **When** a coordenação consulta, **Then** o sistema mostra a necessidade por turma (fluxo Coordenação → RH).

### Épico 6 — Gerencial (Direção)

- **US-601 (Must)** — Como **direção**, quero **ver a inadimplência consolidada**, para **decidir ações de cobrança**.
  - **Given** mensalidades vencidas, **When** a Direção consulta o painel, **Then** vê o total por turma, série e escola, e a evolução mensal (gráfico Chart.js).

- **US-602 (Must)** — Como **direção**, quero **ver alunos por turma e por série**, para **planejar vagas e contratações**.
  - **Given** turmas do ano letivo, **When** consultadas, **Then** o painel mostra ocupação por turma/série.

- **US-603 (Must)** — Como **direção**, quero **ver os gastos por categoria (fornecedores, folha, material)**, para **controlar o orçamento**.
  - **Given** pagamentos registrados, **When** consultados, **Then** o painel mostra gastos por categoria e período.

- **US-604 (Must)** — Como **direção**, quero **ver o estado do estoque**, para **antecipar compras críticas**.
  - **Given** itens abaixo do mínimo, **When** consultados, **Then** aparecem destacados no painel.

- **US-605 (Must)** — Como **direção**, quero **exportar relatórios em PDF**, para **arquivar e compartilhar**.
  - **Given** indicador consultado, **When** exportado, **Then** o PDF é gerado na requisição e a exportação é registrada na auditoria (RS-010).

### Épico 7 — Portal do responsável/aluno (consulta)

- **US-701 (Should)** — Como **responsável**, quero **consultar a situação financeira do meu filho**, para **saber o que está pendente**.
  - **Given** responsável autenticado, **When** consulta, **Then** vê **somente** os dados do próprio filho (RP-006).
  - **Segurança:** escopo `portal:consulta`; autorização no backend verifica o vínculo responsável↔aluno (privilégio mínimo; negar por padrão).

- **US-702 (Should)** — Como **responsável**, quero **consultar o boletim do meu filho**, para **acompanhar o desempenho**.
  - **Given** notas lançadas, **When** consultadas, **Then** o boletim aparece em leitura (sem download de histórico completo no MVP).

- **US-703 (Could)** — Como **responsável**, quero **ver avisos internos da escola**, para **ser informado**.
  - **Given** escola publica aviso interno, **When** o responsável consulta, **Then** o aviso aparece no portal (tabela de notificações internas).

## 5. Priorização (MoSCoW)

| Prioridade | Histórias |
| :--- | :--- |
| **Must** (MVP) | US-001..005 (fundação e acesso); US-101..103; US-201..203; US-301..305; US-401..404; US-501..502; US-601..605 |
| **Should** (MVP, mas pode ceder se prazos apertarem) | US-104, US-105, US-204, US-503, US-701, US-702 |
| **Could** (desejável, não bloqueia o MVP) | US-703 |
| **Won't** (neste release) | Itens da seção 3.2 (fora do MVP) |

> **Justificativa da ordem:** a fundação (Épico 0) vem **antes** de qualquer funcionalidade de negócio — nenhuma tela de módulo existe sem sessão, TOTP, RBAC e auditoria (Diretiva 1). Os épicos de negócio seguem a dependência do fluxograma: Acadêmico → Financeiro (matrícula gera mensalidades), Acadêmico → Pedagógico (matriculados alimentam turmas), Compras/Estoque → Financeiro, RH → Financeiro, e o Gerencial lê todos.

## 6. Premissas, riscos e dependências

### 6.1 Premissas

| ID | Premissa | Impacto se cair |
| :--- | :--- | :--- |
| A-01 | Portal do responsável é consulta somente leitura | Escopo do Épico 7 muda |
| A-02 | Usuários têm navegador moderno; sem app nativo | RNF-011 (responsividade) muda |
| A-03 | Escola de pequeno/médio porte (centenas de alunos) | Metas de RNF e estimativas de cota mudam |
| A-04 | Boleto é demonstrativo, sem registro bancário | Fluxo Financeiro muda (sem baixa automática) |
| A-05 | Dados de demonstração são 100% fictícios; nenhum dado real de aluno/responsável entra no projeto | Violação LGPD — risco de projeto (R-06) |
| A-06 | Um único ano letivo ativo por vez na janela de uso do MVP | Regras de matrícula/rematrícula simplificadas |

### 6.2 Riscos

| ID | Risco | Prob. | Impacto | Mitigação |
| :--- | :--- | :--- | :--- | :--- |
| R-01 | Limites dos planos gratuitos mudam (Vercel Hobby, Neon Free, GitHub Actions) | Alta | Alto | Seção 7 monitora limites; sinalizar na Etapa 2; plano B em ADR para hospedagem |
| R-02 | Vercel Services em beta muda de forma incompatível | Média | Alto | Plano B documentado em ADR (Render + Cloudflare Worker); não implementado por padrão |
| R-03 | Modelos open source erram mais que modelos grandes; tarefas falham ou geram código inseguro | Alta | Médio | Tarefas pequenas e explícitas; verificações automáticas como gates do CI; código gerado tratado como não confiável até passar (Diretiva 4) |
| R-04 | *Slopsquatting*: modelo inventa pacote inexistente e o instalador baixa um malicioso | Média | Alto | Lockfiles versionados, versões fixadas, dependências só da stack aprovada (Diretiva 2); qualquer dependência nova exige ADR + aprovação; OSV-Scanner e Gitleaks no CI |
| R-05 | Prompt injection via issues, comentários de PR ou documentação externa | Média | Alto | Agentes tratam conteúdo externo como dados; só `AGENTS.md`, `ROADMAP.md` e prompt da tarefa definem o que fazer; agentes sem acesso a secrets de produção |
| R-06 | Dado real de aluno/responsável entra em prompt, seed ou demo | Baixo | Crítico | Regra de "somente dados fictícios" em `AGENTS.md` e no SRS (RP-008); revisão humana em cada PR |
| R-07 | PRs maliciosos de forks em repositório público | Média | Alto | Proteção contra deploy de forks ligada na Vercel; workflows sem `pull_request_target`; secrets não expostos a PRs de fork |
| R-08 | Perda do autenticador TOTP por administrador | Baixo | Alto | Procedimento de recuperação no SRS (US-005, RS-008) |
| R-09 | Contexto limitado do modelo executa tarefa grande mal | Alta | Médio | Tarefas do roadmap pequenas o bastante para uma sessão única, com contexto mínimo autocontido |
| R-10 | Cotas de agente de IA gratuitas acabam no meio do projeto | Média | Baixo | Agente alternativo (Freebuff, Kilo Code) definido na stack; tarefas independentes entre si |

### 6.3 Dependências

- **Serviços:** GitHub (repo público, Actions, secrets), Vercel (Hobby, região São Paulo — a confirmar, ver Etapa 2), Neon (Free, `aws-sa-east-1`).
- **Contas:** GitHub pessoal, Vercel, Neon, agente de IA (OpenCode Zen ou equivalente gratuito).
- **Stack:** fixada no prompt e registrada nos ADRs `adr/0001`–`adr/0004`; **não reaberta**.
- **Documentos:** `SRS.md` (requisitos), `THREAT-MODEL.md` (ameaças), `ROADMAP.md` (tarefas para agentes), `TUTORIAL.md` (material acadêmico).

## 7. Viabilidade da abordagem 100% gratuita

> Os limites abaixo são os conhecidos na redação deste documento e **precisam ser revalidados na Etapa 2** — planos gratuitos mudam com frequência (Diretiva 3). Itens não confirmáveis estão marcados com `[VERIFICAR]` e listados nas "Pendências de verificação" no fim deste documento.

| Serviço | Plano | Limites relevantes (conhecidos / a confirmar) | Cabe no MVP? |
| :--- | :--- | :--- | :--- |
| **Vercel** | Hobby (pessoal, não comercial) | Gratuito, sem cartão; sem compra de uso extra (sem risco de cobrança); funções com Fluid compute; região fixável em `gru1` (São Paulo) — `[VERIFICAR]`; Services em beta — `[VERIFICAR]`; limite de builds de preview por dia — `[VERIFICAR]`; proteção de deployment e proteção contra forks disponíveis no Hobby — `[VERIFICAR]` | Sim, com monitoramento |
| **Neon** | Free | PostgreSQL serverless, região `aws-sa-east-1`; branches limitadas no Free (produção + preview = mínimo viável); CU-horas mensais; projeto Free inativo por 90 dias é removido (mitigado pelo backup semanal no Actions) — todos `[VERIFICAR]` na Etapa 2 | Sim (2 branches: `production` + `preview`) |
| **GitHub Actions** | Públicos ilimitados? | Runners padrão `ubuntu-latest` para repositórios públicos: minutos gratuitos ilimitados `[VERIFICAR]`; artifacts têm retenção e armazenamento limitados `[VERIFICAR]` | Sim (repo público na conta pessoal) |
| **GitHub Advanced Security** | Público grátis | Secret scanning e push protection disponíveis em repositórios públicos `[VERIFICAR]` | Sim |
| **CodeQL / OSV-Scanner / Gitleaks / zizmor / govulncheck / golangci-lint** | Open source | Rodam nos runners do Actions; sem custo adicional | Sim |
| **Agentes de IA** | OpenCode Zen / Freebuff / Kilo Code | Catálogo de modelos gratuitos muda a cada poucas semanas; a regra é fixa (modelo open source gratuito, preferindo sem treino sobre os dados), não o nome `[VERIFICAR]` | Sim, com alternativa (Diretiva 4) |
| **Bibliotecas da stack** | Open source | Go, chi, oapi-codegen, sqlc, pgx, goose, scs, argon2id, pquerna/otp, secretbox, httprate, unrolled/secure, caarlos0/env, maroto, brdoc; Vue 3, Vite, Pinia, TanStack Query, Orval, Zod, PrimeVue, Tailwind, pnpm; Playwright, Vitest — todas open source `[VERIFICAR]` licença de cada uma na criação do repo | Sim |
| **OWASP ZAP, Lighthouse CI, axe-core** | Open source | Rodam no CI; sem custo | Sim |

**Conclusão preliminar:** a abordagem é viável sob as diretrizes, condicionada à confirmação dos itens `[VERIFICAR]` na Etapa 2 e ao plano B de hospedagem (ADR-0004).

## 8. Viabilidade do desenvolvimento sem escrita humana de código

**Tese:** a IA conduz o fluxo inteiro de criação do software; humanos especificam, revisam e aprovam.

| # | Risco | Mitigação |
| :--- | :--- | :--- |
| V-01 | Modelos open source menores erram mais (compilação, tipos, testes) | Maximizar verificações automáticas (compilador, type check, lint, testes, varreduras) — o que puder ser gerado por ferramenta determinística (oapi-codegen, sqlc, Orval) não é escrito pela IA; comando de verificação local no Taskfile antes de todo PR |
| V-02 | Contexto limitado: tarefa grande falha | Tarefas pequenas, sequenciais, com contexto mínimo autocontido (ROADMAP, Etapa 4) |
| V-03 | Código inseguro gerado por IA | Código tratado como **não confiável até passar pelas verificações**; gates de segurança bloqueiam merge; revisão humana por PR; nenhum PR revisado pelo mesmo modelo que o escreveu |
| V-04 | Modelo inventa dependência (*slopsquatting*) | Diretiva 2: dependência fora da stack exige ADR + aprovação; lockfiles e versões fixadas; OSV-Scanner |
| V-05 | Modelo ignora requisitos de segurança | Requisitos com ID rastreável; critérios de aceite de segurança por história; matriz de rastreabilidade |
| V-06 | Falta de memória entre sessões | `AGENTS.md` como fonte de regras; ID da tarefa e modelo registrados no corpo do PR; documentos autocontidos |
| V-07 | Humanos sem tempo/reconhecimento para revisar PRs de IA | Tutorial (TUTORIAL.md) ensina a revisar PR, inclusive via preview; Rastreabilidade no PR (ID da tarefa, agente e modelo) |
| V-08 | Modelos inconsistentes entre si (OpenCode vs Freebuff vs Kilo Code) | Tarefas não dependem de recursos exclusivos de uma ferramenta; `AGENTS.md` na raiz vale para todas |

**Conclusão preliminar:** viável, com as mitigations acima; o risco residual maior é V-01/V-02 (qualidade do código gerado), mitigado por gates automáticos e revisão humana obrigatória por PR.

## 9. Matriz de rastreabilidade (resumo do PRD)

A matriz completa (histórias × requisitos × ameaças × módulos) vive no SRS. Aqui, o resumo por épico:

| Épico / Histórias | Requisitos funcionais (SRS) | Ameaças (THREAT-MODEL) | Módulo |
| :--- | :--- | :--- | :--- |
| US-001..005 (Épico 0) | RF-001..009, RS-001..009 | AM-001..003, AM-010, AM-011 | Transversal (sessão, TOTP, RBAC, auditoria) |
| US-101..105 (Épico 1) | RF-101..108, RN-001..003 | AM-004, AM-006, AM-012 | Acadêmico |
| US-201..204 (Épico 2) | RF-201..208, RN-004..005 | AM-005, AM-007 | Pedagógico |
| US-301..305 (Épico 3) | RF-301..309, RN-001, RN-007 | AM-006, AM-008 | Financeiro |
| US-401..404 (Épico 4) | RF-401..407, RN-006 | AM-008 | Compras e Estoque |
| US-501..503 (Épico 5) | RF-501..505 | AM-009 | RH |
| US-601..605 (Épico 6) | RF-601..606 | AM-006 | Gerencial |
| US-701..703 (Épico 7) | RF-701..703, RP-006 | AM-006, AM-012 | Portal do responsável |

## 10. Pendências de verificação `[VERIFICAR]`

Itens não confirmáveis na redação (sem acesso à internet no momento) — a revalidar na Etapa 2:

1. Vercel Hobby: fixar região das funções em `gru1` (São Paulo).
2. Vercel Services (modelo atual `services` + rewrites): disponibilidade e comportamento no plano Hobby; interação com `vercel build` + `vercel deploy --prebuilt`.
3. Vercel Hobby: limite de builds de preview por dia; proteção de deployment padrão e proteção contra deploy de forks.
4. Convenção de entrypoint/porta do formato zero-config de backend Go da Vercel (confirmar na documentação no momento da implementação).
5. Desligar o deploy automático de produção pela integração Git via configuração `git` do `vercel.json` (confirmar chave exata).
6. Neon Free: CU-horas mensais, número de branches, limpeza de projetos inativos após 90 dias, região `aws-sa-east-1`.
7. GitHub Actions: minutos ilimitados em runners padrão para repositórios públicos; limites de armazenamento/retensão de artifacts.
8. GitHub: secret scanning e push protection disponíveis (e ligados) em repositórios públicos.
9. Regras do firewall da Vercel disponíveis no plano Hobby (camada extra de rate limiting).
10. Catálogo atual de modelos gratuitos no OpenCode Zen (e alternativas Freebuff/Kilo Code) — a regra é fixa, não o nome.
11. Licenças open source de todas as bibliotecas da stack, a conferir na criação do repositório.
