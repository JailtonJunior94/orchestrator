---
name: domain-modeling-production
version: 1.0.0
description: Conduz modelagem de dominio orientada a producao em PT-BR, transformando problema de negocio, linguagem ubiqua, regras, invariantes, comandos, eventos, estados e fronteiras em um modelo explicito, economico, robusto e implementavel. Use para discovery de dominio, desenho de workflows, definicao de agregados, contratos, politicas e erros de negocio antes de codigo, API ou backlog. Nao use para brainstorming solto, refinamento apenas de UX, implementacao direta de codigo ou arquitetura de infraestrutura sem foco no dominio.
---

# Modelagem de Dominio Production-Ready

<critical>Todos os artefatos DEVEM ser escritos em PT-BR.</critical>
<critical>Modelar comportamento, regras e decisoes antes de modelar tabela, endpoint, fila ou classe.</critical>
<critical>Estados ilegais DEVEM ser bloqueados explicitamente por invariantes, regras de transicao, erros de dominio ou combinacao desses mecanismos.</critical>
<critical>Comandos, eventos, estados, politicas e erros de negocio DEVEM ser modelados separadamente. Nao colapsar tudo em CRUD, status ou campos soltos.</critical>
<critical>Quando houver codebase existente, compatibilidade confirmada DEVE citar evidencia `path:linha`. Match textual isolado, teste, mock, fixture, exemplo, documentacao ou arquivo gerado nao confirmam uso em producao.</critical>
<critical>Economia, eficiencia e robustez sao restricoes de primeira classe. A skill DEVE preferir modelo simples, correto e operacionalmente sustentavel a uma taxonomia elegante, mas cara ou fraca.</critical>
<critical>Nao materializar o modelo enquanto existir ambiguidade material em linguagem ubiqua, regra critica, ownership, fronteira, invariantes, erro de negocio ou trade-off operacional relevante.</critical>
<critical>O bundle so e considerado pronto quando `scripts/validate-bundle.py` retornar `SUCCESS`.</critical>
<critical>O comportamento da skill DEVE ser agnostico de agente: Claude Code, Codex, Gemini e Copilot DEVEM seguir a mesma ordem de passos, os mesmos gates, os mesmos artefatos e a mesma politica de clarificacao.</critical>

## Entrada Obrigatoria
- Problema, fluxo ou capacidade de negocio a ser modelado.
- Objetivo da modelagem: esclarecer regra, desenhar fluxo, preparar implementacao, revisar dominio existente ou reduzir ambiguidade antes de backlog/codigo.

## Entrada Recomendavel
- Linguagem ja usada pelo negocio, regras, politicas, edge cases, restricoes regulatorias, KPIs, incidentes ou reclamacoes.
- Escopo de codebase para confronto: path local, modulo, repo remoto `owner/repo`, servico, pacote ou declaracao explicita de greenfield.
- Artefatos existentes: PRD, RFC, API spec, schema, BPMN, playbook operacional, queries, eventos, planilhas ou tickets.

## Saida
Bundle local em `./discoveries/domain-<slug>/`:
- `bundle.json` - indice do bundle com status, titulo, prontidao e resumos estruturais.
- `domain-model.md` - artefato principal com linguagem ubiqua, workflows, tipos conceituais, regras e fronteiras.
- `transcript.md` - historico auditavel das rodadas, evidencias e decisoes.

## Contrato de Compatibilidade
- Nao depender de componente visual, widget, formulario ou API especifica de um agente.
- Quando houver pergunta estruturada, usar esse mecanismo sem alterar o conteudo logico das opcoes.
- Quando nao houver pergunta estruturada, usar texto puro com opcoes `A`, `B`, `C`, `D` e aguardar resposta inequivoca.
- Nao gerar codigo, migrations, backlog, PRD, TechSpec, SDD ou work items automaticamente.
- Registrar toda decisao efetiva no `transcript.md` e refletir o consolidado em `domain-model.md`.

## Procedimentos

**Step 1: Validar entrada e inicializar bundle**
1. Identificar um titulo curto a partir do pedido do usuario.
2. Executar `python3 scripts/slugify.py "<titulo>"` para normalizar o slug.
3. Verificar se `./discoveries/domain-<slug>/` ja existe. Se existir, perguntar em multipla escolha se deve reaproveitar, criar novo com sufixo ou cancelar.
4. Executar `python3 scripts/init-bundle.py <slug>` para criar `bundle.json`, `domain-model.md` e `transcript.md`.
5. Encerrar com `blocked` se o script falhar por conflito de diretorio, permissao ou slug invalido.

**Step 2: Coletar objetivo, materiais e escopo de codebase**
1. Pedir ao usuario, em multipla escolha, qual e a natureza principal do pedido: novo fluxo, revisao de fluxo existente, correcao de ambiguidade, quebra de dominio mal modelado, integracao entre dominios ou outra categoria equivalente.
2. Pedir ao usuario, em multipla escolha, qual e o estado do material de apoio: regras claras, regras parciais, apenas contexto verbal, ou sistema legado pouco compreendido.
3. Pedir ao usuario, em multipla escolha, qual e o escopo de codebase: path local, repo remoto, greenfield sem codebase existente, ou confronto indisponivel com risco explicito.
4. Solicitar artefatos concretos que sustentam a modelagem: paths, arquivos, links, contratos, exemplos de input/output, eventos, planilhas ou descricoes curtas. Se nao houver materiais, registrar a ausencia.
5. Resumir o entendimento inicial em ate 6 bullets e registrar em `transcript.md` no bloco `## Contexto Inicial`.

**Step 3: Confrontar pedido com codebase e padroes existentes**
1. Ler `references/codebase-confrontation.md`.
2. Se houver path local, buscar termos do dominio, nomes de fluxo, identificadores, eventos, tabelas, enums, erros, validacoes, policies e mapeamentos externos.
3. Se houver repo remoto, usar `gh` apenas quando disponivel e autenticado; caso contrario, registrar bloqueio ou risco conforme criticidade.
4. Classificar cada achado como `confirmado`, `suspeito`, `ausente`, `refutado` ou `greenfield`.
5. Para evidencia `confirmado`, guardar `path:linha` e observacao curta. Evidencia em teste, mock, fixture, exemplo, documentacao ou arquivo gerado fica como `suspeito` por padrao.
6. Registrar resumo em `transcript.md` no bloco `## Confronto com Codebase`.

**Step 4: Rodada 1 obrigatoria - linguagem ubiqua, resultado e fronteiras**
1. Ler `references/clarification-rounds.md` e aplicar os eixos da Rodada 1.
2. Formular de 3 a 4 perguntas em multipla escolha cobrindo, no minimo: objetivo de negocio dominante, termo canonico do fluxo, bounded context ou fronteira principal, e restricao dominante.
3. Em cada pergunta, explicitar a consequencia operacional, de custo, robustez ou risco de escolher cada opcao.
4. Registrar perguntas, opcoes e respostas em `transcript.md` no bloco `## Rodada 1 - Linguagem e Fronteiras`.
5. Atualizar o rascunho interno do modelo com termos proibidos, sinonimos e ownership preliminar.

**Step 5: Rodada 2 obrigatoria - workflow, comandos, eventos e estados**
1. Ler `references/clarification-rounds.md` e aplicar os eixos da Rodada 2.
2. Formular de 3 a 4 perguntas cobrindo, no minimo: gatilho do fluxo, comando principal, evento relevante, estado ou transicao critica e ponto de falha relevante.
3. Nao aceitar modelagem puramente CRUD quando houver decisao de negocio, regra condicional, validacao ou consequencia operacional real.
4. Registrar tudo em `transcript.md` no bloco `## Rodada 2 - Workflow e Comportamento`.

**Step 6: Rodada 3 obrigatoria - regras, invariantes, politicas e erros**
1. Ler `references/clarification-rounds.md` e aplicar os eixos da Rodada 3.
2. Formular de 3 a 4 perguntas cobrindo, no minimo: regra de negocio central, invariante nao negociavel, politica/decisao calculada, e estrategia de erro de dominio.
3. Se o usuario nao souber nomear a regra, oferecer opcoes conservadoras que permitam explicitar a restricao sem inventar semantica fraca.
4. Registrar tudo em `transcript.md` no bloco `## Rodada 3 - Regras e Invariantes`.

**Step 7: Rodada 4 obrigatoria - tipos conceituais, integracoes, custo e operacao**
1. Ler `references/clarification-rounds.md` e aplicar os eixos da Rodada 4.
2. Formular de 3 a 4 perguntas cobrindo, no minimo: agregado ou ownership transacional, fronteiras externas, consistencia/persistencia e postura de custo/operacao.
3. Confrontar cada opcao com robustez, simplicidade, compatibilidade com codebase e custo de manutencao.
4. Registrar tudo em `transcript.md` no bloco `## Rodada 4 - Tipos e Integracoes`.

**Step 8: Abrir rodadas adicionais enquanto houver risco material**
1. Ler `references/quality-gates.md`.
2. Avaliar se ainda faltam definicoes para qualquer gate obrigatorio: linguagem ubiqua, workflow principal, comandos, eventos, invariantes, erros, fronteiras, ownership, compatibilidade, observabilidade ou custo.
3. Se faltar, abrir Rodada 5+ com perguntas focadas exclusivamente nos pontos pendentes.
4. Manter cada rodada com no maximo 4 perguntas e registrar tudo no `transcript.md`.
5. Nao materializar `domain-model.md` enquanto existir bloqueio material nao decidido.

**Step 9: Consolidar direcao do modelo**
1. Apresentar ao usuario um resumo consolidado em ate 10 bullets com: problema, objetivo, bounded contexts, fluxo principal, comandos, eventos, invariantes, erros, ownership, fronteiras externas e trade-offs aceitos.
2. Perguntar em multipla escolha se deve: materializar o modelo agora, refinar um ponto especifico, ou cancelar.
3. Se o usuario pedir refinamento, voltar ao Step 8.
4. Se o usuario cancelar, encerrar com `done` sem materializar novos artefatos alem do transcript.

**Step 10: Materializar o modelo de dominio**
1. Ler `assets/domain-model-template.md`.
2. Ler `references/modeling-principles.md`.
3. Preencher `domain-model.md` integralmente com base nas respostas e evidencias coletadas. Nao inventar fatos ausentes; registrar lacunas explicitamente em `## Itens em Aberto`.
4. Garantir que as secoes de linguagem ubiqua, bounded contexts, workflow, comandos, eventos, invariantes, tipos conceituais, erros, fronteiras, observabilidade, custo e decisoes abertas sejam especificas ao contexto do usuario, sem texto generico reaproveitado.
5. Atualizar `bundle.json` com titulo, status, contextos, principais decisoes e blockers remanescentes.

**Step 11: Validar e corrigir**
1. Executar `python3 scripts/validate-bundle.py ./discoveries/domain-<slug>`.
2. Se houver erro, ler o stderr, identificar a secao, corrigir o documento e reexecutar.
3. Encerrar com `blocked` se a validacao continuar falhando apos uma rodada honesta de correcao.

**Step 12: Relatar a saida**
1. Informar o caminho do bundle gerado.
2. Resumir em ate 6 bullets os pontos centrais do modelo, regras mais sensiveis, riscos residuais e prontidao para handoff.
3. Sugerir o proximo passo conforme maturidade: `technical-discovery-production` para detalhamento tecnico maior, `epic-story-discovery` para backlog, ou implementacao guiada quando o modelo ja estiver suficientemente estavel.
4. Nao executar a proxima skill automaticamente.

## Decisoes Operacionais
1. Tratar o dominio como mecanismo de decisao, nao como lista de campos.
2. Preferir tipos conceituais pequenos e composicionais a um agregado monolitico sem fronteira clara.
3. Separar modelo interno de contratos externos, DTOs, schemas, filas e persistencia.
4. Quando o mesmo termo tiver significados diferentes, nomear a diferenca explicitamente em vez de esconder o conflito.
5. Quando um workflow puder falhar de modo relevante, modelar o erro de negocio, nao apenas uma mensagem generica.
6. Preferir premissa explicita e auditavel a inferencia forte.
7. Introduzir novo bounded context, agregado ou evento apenas quando houver ganho defensavel de robustez, isolamento, clareza ou operacao.
8. Preservar terminologia do negocio e nomes dos sistemas informados pelo usuario.
9. Tratar custo cognitivo, custo operacional e custo de mudanca como parte do design do dominio.
10. Se nao houver ferramenta de pergunta estruturada, manter multipla escolha em texto e aguardar resposta antes de seguir.

## Estados Finais
- `done`: transcript e, quando aprovado, modelo materializados e validados.
- `needs_input`: falta resposta do usuario para risco material ou falta insumo indispensavel apos tentativa de clarificacao.
- `blocked`: erro de I/O, conflito de diretorio, falha persistente de validacao ou impossibilidade de materializar o bundle.
- `failed`: erro inesperado de execucao apos tentativa de recuperacao.

## Tratamento de Erros
- Se `scripts/slugify.py` retornar slug vazio, pedir um titulo curto ao usuario e repetir a normalizacao.
- Se `scripts/init-bundle.py` falhar por diretorio existente, oferecer em multipla escolha reaproveitar, versionar ou cancelar.
- Se o usuario insistir em seguir sem materiais de apoio, registrar isso em `## Materiais e Evidencias`, marcar risco correspondente e abrir perguntas adicionais de reducao de incerteza.
- Se a validacao apontar placeholder nao resolvido em secao critica, completar a secao ou registrar a lacuna em `## Itens em Aberto` e voltar para nova rodada antes de revalidar.
- Se nao for possivel definir linguagem ubiqua, invariante central, ownership, erro de dominio, fronteira externa ou postura de custo/operacao com o material disponivel, encerrar com `needs_input` em vez de declarar o modelo pronto para uso.
