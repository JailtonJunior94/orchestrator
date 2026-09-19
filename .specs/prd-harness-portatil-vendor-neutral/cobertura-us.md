# Cobertura da User Story — auditoria item a item

<!-- Artefato de rastreabilidade do PRD `prd-harness-portatil-vendor-neutral`. -->

Auditoria dos **138 itens verificáveis** da User Story `US-harness-portatil-claude-codex-opencode.md`:
**111 critérios de aceite** + **27 itens do Definition of Done**. Nenhum item fica sem destino.

Legenda: *Baseline* = já entregue e verificado em `v2.0.1`, preservado por RF-48 (não reespecificado).
**LACUNA FECHADA** = item que a primeira versão deste PRD não cobria e que foi adicionado nesta auditoria.


## RF01 — Fonte canônica única

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Existe uma definição explícita de quais arquivos formam o core canônico. | RF-01 | Inventário classifica cada regra e define o core canônico |
| 2 | Regras universais não precisam ser mantidas manualmente em três locais. | RF-09, RF-13 | Origem única em `.agents/policies/` (planejado); duplicata manual proibida |
| 3 | Configurações específicas de fornecedor são tratadas como adapters. | RF-16 | Config de fornecedor é adapter que referencia o core |
| 4 | Uma alteração em uma regra universal possui uma única origem. | RF-09, RF-11 | Origem única + gate de sincronia |
| 5 | O processo detecta divergência entre conteúdo canônico e conteúdo derivado. | RF-11 | Gate dedicado falha na divergência canônico↔derivado |

## RF02 — Contrato formal do Harness

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | O contrato possui versão explícita. | RF-02 | `version: 1` explícito |
| 2 | O contrato representa políticas de Git. | RF-02 | Bloco de política de Git |
| 3 | O contrato representa regras de aprovação. | RF-02 | Bloco de aprovação |
| 4 | O contrato representa requisitos de teste e validação. | RF-02 | Bloco de qualidade: testes e lint |
| 5 | O contrato representa política de evidências. | RF-02 | Bloco de evidência |
| 6 | O contrato representa política de descoberta de skills. | RF-02 | Bloco de descoberta de skills |
| 7 | Campos desconhecidos ou versões incompatíveis falham de maneira explícita. | RF-03, RF-04 | Campo desconhecido e versão incompatível = erro tipado |
| 8 | Configuração inválida não é silenciosamente ignorada. | RF-03 | Fail-closed; nunca warning |

## RF03 — Instalação em outro projeto

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | A operação identifica a raiz do repositório. | RF-23 | Já entregue: resolve raiz do repositório |
| 2 | Não sobrescreve conteúdo existente sem política explícita. | RF-26, RF-27 | Não gerenciado nunca é sobrescrito; conflito aborta o lote |
| 3 | É idempotente. | RF-23 | Já entregue e testado |
| 4 | Instala somente arquivos necessários. | RF-23 | Já entregue |
| 5 | Informa arquivos criados, atualizados, preservados e em conflito. | RF-23.1 | **LACUNA FECHADA**: hoje só existem criado/mesclado; passa a relatório tipado com 5 categorias |
| 6 | Não executa commit automaticamente. | RF-55 | Verificado por teste, não só afirmado |
| 7 | Não executa push automaticamente. | RF-55 | Idem |
| 8 | Não exige credenciais de Claude, OpenAI ou OpenCode para instalar o harness. | RF-23 | Instalação é operação de arquivos |
| 9 | O projeto permanece utilizável sem o processo do `orchestrator` rodando. | RF-52 | Sem daemon permanente |

## RF04 — Sincronização e atualização

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | A versão instalada é identificável. | RF-29 | Versão instalada identificável |
| 2 | A versão disponível é identificável. | RF-29 | Versão disponível identificável |
| 3 | O usuário consegue visualizar alterações antes de aplicá-las. | RF-25 | Diff antes de aplicar |
| 4 | Arquivos locais não gerenciados não são sobrescritos. | RF-26 | Arquivo não gerenciado preservado |
| 5 | Conflitos são apresentados explicitamente. | RF-24, RF-27 | Conflito é categoria explícita, detectada por `path → checksum` |
| 6 | O processo é determinístico. | RF-29 | Mesma origem e destino = mesmo resultado |
| 7 | Falha parcial não deixa o harness em estado silenciosamente inconsistente. | RF-28 | Transacional: tudo ou nada; reverte e reporta |
| 8 | Existe mecanismo de validação após atualização. | RF-29 | Validação pós-atualização |
| 9 | Nenhum commit ou push ocorre implicitamente. | RF-55 | Nenhum commit/push implícito, testado |

## RF05 — Skills compartilhadas

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Uma mesma skill pode ser utilizada pelas três ferramentas quando tecnicamente suportado. | RF-48 | Baseline: skills compartilhadas pelos 4 provedores |
| 2 | Skills possuem metadata suficiente para descoberta. | RF-48 | Baseline: frontmatter com metadata de descoberta |
| 3 | O conteúdo completo não precisa ser carregado quando a skill não for necessária. | RF-42 | Carregamento sob demanda |
| 4 | Skills específicas de fornecedor são explicitamente identificadas. | RF-21 | Skill de fornecedor é classificada explicitamente |
| 5 | Uma skill exclusiva de um fornecedor não é apresentada como capability universal. | RF-21 | Proibido apresentar exclusiva como universal |
| 6 | O mecanismo existente de integridade/versionamento das skills continua válido. | RF-31 | Integridade por hash verificada no bloco Core do doctor |
| 7 | `skills-lock.json`, ou mecanismo equivalente existente, permanece verificável. | RF-31, RF-48 | `skills-lock.json` permanece verificável e intocado |

## RF06 — Progressive disclosure

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | O arquivo de entrada contém somente instruções universais necessárias. | RF-42 | Só instruções universais na entrada |
| 2 | Documentação especializada é carregada sob demanda. | RF-42 | Doc especializada sob demanda |
| 3 | Skills não relacionadas à tarefa não precisam entrar no contexto. | RF-42 | Skill irrelevante fora do contexto |
| 4 | Policies extensas podem ser referenciadas sem duplicação. | RF-10, RF-42 | Policy referenciada a partir da origem única, sem duplicar |
| 5 | Existe avaliação automatizada ou benchmark capaz de detectar crescimento desnecessário do contexto. | RF-41, RF-41.1 | Budget de contexto de entrada por provedor, baseline commitado, gate em CI |

## RF07 — Adapter Claude Code

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Claude Code consegue descobrir as instruções necessárias ao abrir o projeto. | RF-48 | Baseline: Claude descobre instruções ao abrir |
| 2 | O adapter referencia o core canônico em vez de manter uma cópia completa. | RF-16 | Adapter referencia o core |
| 3 | Skills compartilháveis permanecem na estrutura canônica. | RF-48 | Skills na estrutura canônica |
| 4 | Recursos exclusivos do Claude ficam isolados na configuração específica. | RF-16 | Exclusivos isolados na config específica |
| 5 | O adapter não altera invariantes universais. | RF-17 | Gate falha se adapter enfraquecer política obrigatória |
| 6 | Existe teste de compatibilidade específico para Claude Code. | RF-36, RF-37 | Compatibilidade coberta nos dois níveis |

## RF08 — Adapter Codex

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Codex consegue utilizar `AGENTS.md` do projeto. | RF-48 | Baseline: Codex usa `AGENTS.md` |
| 2 | Skills compartilhadas são disponibilizadas pelo diretório canônico suportado. | RF-48 | Skills no diretório canônico |
| 3 | Configuração específica do Codex existe somente quando necessária. | RF-16 | Config específica só quando necessária |
| 4 | Nenhuma regra universal é duplicada desnecessariamente. | RF-13, RF-16 | Zero duplicação de regra universal |
| 5 | Existe teste de compatibilidade específico para Codex. | RF-36, RF-37 | Teste de compatibilidade Codex |

## RF09 — Adapter OpenCode

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | OpenCode consegue acessar as instruções universais. | RF-48 | Baseline (ADR-020): OpenCode acessa instruções universais |
| 2 | Skills compartilháveis utilizam a estrutura canônica quando suportada. | RF-48 | Estrutura canônica |
| 3 | Configuração própria do OpenCode apenas complementa o core. | RF-16 | Config complementa o core |
| 4 | Não existe fork independente das regras do harness. | RF-13, RF-16 | Sem fork de regras |
| 5 | Existe teste de compatibilidade específico para OpenCode. | RF-36, RF-37 | Teste de compatibilidade OpenCode |

## RF10 — Capability Matrix

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | A matriz é versionada junto ao código. | RF-18, RF-19 | JSON + Markdown commitados, gate de sincronia |
| 2 | Não existe capability universal marcada como suportada sem teste correspondente. | RF-20, RF-18.1 | Célula sem teste falha; invariante vacuoso saneado antes |
| 3 | Diferenças de comportamento são documentadas. | RF-18, RF-21 | Nível de suporte e diferenças declarados por célula |
| 4 | Capability não suportada falha ou é reportada explicitamente. | RF-21 | Capability não suportada falha ou é reportada |
| 5 | Nenhum adapter pode enfraquecer silenciosamente uma política obrigatória. | RF-17 | Gate falha em enfraquecimento silencioso |

## RF11 — Política de Git

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Claude respeita a política. | RF-40 | Claude respeita, provado nos dois níveis |
| 2 | Codex respeita a política. | RF-40 | Codex idem |
| 3 | OpenCode respeita a política. | RF-40 | OpenCode idem |
| 4 | Existe teste adversarial para tentativa de auto-commit. | RF-40.1 | **LACUNA FECHADA**: enforcement de auto-commit não existe hoje; este PRD o cria |
| 5 | Existe teste adversarial para operação destrutiva. | RF-40.2 | **LACUNA FECHADA**: destrutividade separada de preload |
| 6 | A ausência de confirmação necessária resulta em bloqueio seguro. | RF-40 | Bloqueio seguro na ausência de confirmação |

## RF12 — Approval Gates

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Operações classificadas como destrutivas exigem aprovação. | RF-48 | Baseline: aprovação para destrutivas |
| 2 | Estados de aprovação inválidos são rejeitados. | RF-48 | Baseline: estado inválido rejeitado |
| 3 | Falta de evidência obrigatória impede aprovação. | RF-48 | Baseline: sem evidência não aprova |
| 4 | O comportamento é fail-closed quando a decisão não puder ser comprovada. | RF-48 | Baseline: fail-closed |
| 5 | As três CLIs são avaliadas contra os mesmos cenários. | RF-38 | Mesmo fixture avalia os 4 provedores |

## RF13 — Evidence

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Evidências possuem formato estruturado. | RF-48 | Baseline: formato estruturado |
| 2 | Evidências podem ser validadas independentemente do texto final do LLM. | RF-48 | Baseline: validação independente do texto do LLM |
| 3 | Resultado declarado pelo agente não substitui evidência obrigatória. | RF-48 | Baseline: declaração não substitui evidência |
| 4 | Evidências inválidas ou ausentes são detectadas. | RF-48 | Baseline: ausência detectada |
| 5 | O formato não depende exclusivamente de Claude, Codex ou OpenCode. | RF-48 | Baseline: formato tool-neutro |

## RF14 — Deterministic Gates

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | O harness não utiliza LLM para validar algo comprovável deterministicamente sem justificativa. | RF-08, RF-34 | Verificação estática sem LLM |
| 2 | Falha determinística impede aprovação indevida. | RF-48 | Baseline: falha determinística impede aprovação |
| 3 | Gates possuem saída reproduzível. | RF-38 | Saída reproduzível no que é determinístico |
| 4 | Resultado é incorporado às evidências. | RF-48 | Baseline: resultado entra na evidência |

## RF15 — SDD compartilhado

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Artefatos SDD possuem formato independente de fornecedor. | RF-48 | Baseline: artefatos SDD tool-neutros |
| 2 | Claude consegue continuar um fluxo criado por Codex. | RF-36 | Continuidade cross-provider entra como cenário determinístico de CI |
| 3 | Codex consegue continuar um fluxo criado por Claude. | RF-36 | Idem |
| 4 | OpenCode consegue continuar um fluxo criado pelos demais. | RF-36 | Idem |
| 5 | Estado persistido não depende do histórico interno de uma CLI específica. | RF-36 | Estado persiste em disco, não em sessão |
| 6 | Retomar uma tarefa não exige obrigatoriamente a mesma ferramenta que iniciou a execução. | RF-36 | Retomada não exige a mesma ferramenta |

## RF16 — Doctor multi-provider

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Diagnóstico separa core de adapters. | RF-30 | Blocos Core + um por provedor |
| 2 | Detecta arquivo obrigatório ausente. | RF-31, RF-33 | Arquivo obrigatório ausente |
| 3 | Detecta configuração divergente. | RF-33 | Configuração divergente |
| 4 | Detecta versão incompatível. | RF-33 | Versão de contrato incompatível |
| 5 | Detecta integridade inválida de skill quando aplicável. | RF-31, RF-33 | Integridade de skill inválida |
| 6 | Não exige chamada paga a LLM para verificações estáticas. | RF-34 | Só verificação estática |
| 7 | Retorna exit code não zero quando invariantes obrigatórios falham. | RF-34 | Exit code ≠ 0 em invariante obrigatório |

## RF17 — Suite de conformidade cross-provider

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | O mesmo fixture pode avaliar os três providers. | RF-38 | Mesmo fixture, 4 provedores |
| 2 | A avaliação mede invariantes, não igualdade textual da resposta. | RF-38 | Mede invariantes, nunca igualdade textual |
| 3 | Resultados são reproduzíveis naquilo que é determinístico. | RF-38 | Reprodutível no determinístico |
| 4 | Falhas são atribuídas ao core, adapter ou provider quando possível. | RF-39 | Atribuição a core/adapter/provider |
| 5 | Regressões bloqueiam release conforme política definida. | RF-56 | **LACUNA FECHADA**: política de release declarada explicitamente |

## RF18 — Ablation Tests

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Existe mecanismo de benchmark baseline. | RF-47 | Mecanismo de baseline |
| 2 | Componentes caros podem ser avaliados isoladamente. | RF-47 | Componente avaliado isoladamente |
| 3 | A decisão KEEP / ON-DEMAND / REMOVE é sustentada por evidência. | RF-47 | KEEP/ON-DEMAND/REMOVE por evidência registrada |
| 4 | Métricas não dependem apenas de avaliação subjetiva do LLM. | RF-45, RF-47 | Métricas objetivas, nunca julgamento subjetivo do LLM |

## RF19 — Métricas comparáveis

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Ausência de métrica não impede execução quando o provider não a disponibilizar. | RF-45 | Ausência não impede execução |
| 2 | Métricas ausentes são representadas como desconhecidas, nunca inventadas. | RF-45 | Ausente é ausente, nunca inventada |
| 3 | Dados de Claude, Codex e OpenCode usam schema comum. | RF-44 | Conjunto comum de chaves para os 4 |
| 4 | Métricas específicas de provider ficam em extensão própria. | RF-45 | Extensão própria por provedor |
| 5 | Nenhuma chave secreta ou conteúdo sensível é persistido por telemetria. | RF-46 | Teste falha se campo sensível for emitido |

## Definition of Done

| # | Critério da US | Destino no PRD | Observação |
|---|----------------|----------------|------------|
| 1 | Existe uma única fonte canônica para regras universais. | RF-09 | Origem canônica única |
| 2 | Harness Contract v1 está implementado e versionado. | RF-02 | Contrato v1 versionado |
| 3 | Claude, Codex e OpenCode possuem adapters mínimos. | RF-15, RF-16 | Adapters mínimos, 4 provedores |
| 4 | O mesmo projeto pode alternar entre as três CLIs. | RF-53 | **LACUNA FECHADA**: alternância entre CLIs coberta por teste |
| 5 | Não é necessário reinstalar o harness para trocar de CLI. | RF-53 | **LACUNA FECHADA**: troca sem reinstalar |
| 6 | Skills universais são compartilhadas. | RF-48 | Skills universais compartilhadas |
| 7 | Policies universais são compartilhadas. | RF-09, RF-12 | Policies universais compartilhadas |
| 8 | Workflows universais são compartilhados. | RF-54 | **LACUNA FECHADA**: respondido por ausência de objeto, com gatilho de reabertura |
| 9 | SDD é independente de provider. | RF-48 | SDD independente de provider |
| 10 | Evidências são independentes de provider. | RF-48 | Evidências independentes de provider |
| 11 | Política de Git é equivalente entre os três. | RF-40 | Política de Git equivalente entre os 4 |
| 12 | Auto-commit permanece desabilitado por padrão. | RF-40.1 | Auto-commit desabilitado por padrão, com enforcement real |
| 13 | Operações destrutivas respeitam approval. | RF-40.2 | Destrutivas respeitam approval |
| 14 | `init` é idempotente. | RF-22, RF-23 | `init` idempotente |
| 15 | Sincronização é segura e detecta conflitos. | RF-24, RF-27, RF-28 | Sync segura, detecta conflito, transacional |
| 16 | `doctor` valida core e os três providers. | RF-30 | `doctor` valida core + 4 provedores |
| 17 | Existe capability matrix automatizável. | RF-18, RF-19 | Capability matrix gerada e versionada |
| 18 | Existe suite de conformidade cross-provider. | RF-35 | Suíte de conformidade cross-provider |
| 19 | Existem testes adversariais para políticas críticas. | RF-40 | Adversariais para políticas críticas |
| 20 | Existe avaliação de progressive disclosure. | RF-41 | Avaliação de progressive disclosure |
| 21 | Regressões relevantes são detectadas antes de release. | RF-48, RF-56 | Regressões detectadas antes do release |
| 22 | Documentação explica instalação, atualização e uso. | RF-51 | Documentação de instalação, atualização e uso |
| 23 | O modo nativo funciona sem daemon permanente do `orchestrator`. | RF-52 | Modo nativo sem daemon |
| 24 | Nenhuma regra crítica depende apenas de o LLM “lembrar” de obedecê-la. | RF-40.1 | Nenhuma regra crítica depende de o LLM lembrar |
| 25 | Não foi introduzida duplicação integral de harness por fornecedor. | RF-13, RF-16 | Sem duplicação integral por fornecedor |
| 26 | `go test ./...` e gates existentes passam. | RF-48 | `go test ./...` e gates existentes passam |
| 27 | Evidências da implementação estão armazenadas conforme o padrão atual do projeto. | RF-57 | **LACUNA FECHADA**: evidências persistidas no padrão vigente |

## Itens fora de escopo declarados pela User Story

Os 14 itens da seção *Fora de Escopo* da US (OpenRouter, agent swarm, multi-model voting, vector
database, RAG obrigatório, graph database, MCP só-leitura, novo planner, auto-commit irrestrito,
auto-push, auto-merge e reimplementação das três CLIs) estão reproduzidos integralmente na seção
*Fora de Escopo* do PRD e nenhum é introduzido por requisito algum deste documento.

## Cenário de Aceite End-to-End

O fluxo `init` → `claude` → `codex` → `opencode` sem reinstalação, com artefatos persistentes
compreendidos entre ferramentas, é coberto por RF-53 (alternância sem reinstalar), RF-36
(continuidade SDD cross-provider) e RF-52 (sem daemon). Os oito selos finais do cenário
(políticas respeitadas, skills corretas, testes executados, gates determinísticos, critérios
verificáveis, evidências produzidas, approval respeitado, nenhum commit/push implícito) mapeiam
respectivamente para RF-40, RF-21, RF-48, RF-08/RF-34, RF-48, RF-57, RF-40.2 e RF-55.
