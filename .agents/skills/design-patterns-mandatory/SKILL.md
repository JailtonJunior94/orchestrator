---
name: design-patterns-mandatory
version: 1.0.0
description: Padroniza a selecao, justificativa, especificacao, refatoracao e validacao dos design patterns classicos do catalogo Refactoring.Guru com foco obrigatorio em economia, eficiencia e robustez, produzindo decisoes deterministicas, pseudocodigo canonico, mapeamento por paradigma e plano de testes para uso em desenvolvimento de software. Use quando o pedido envolver escolher, aplicar, revisar, comparar ou rejeitar padroes de projeto em codigo ou arquitetura de aplicacao. Nao use para documentacao generica, ensino teorico sem contexto de implementacao, ou para recomendar padroes sem evidencia tecnica suficiente.
---

# Design Patterns Mandatory

<critical>Priorizar a solucao mais simples, barata e robusta. Nao aplicar pattern quando codigo direto, funcao simples, modulo pequeno ou refactor localizado resolverem o problema com menor custo total.</critical>
<critical>Usar esta skill apenas com evidencia concreta do problema, do codigo e das restricoes. Sem evidencia suficiente, interromper e pedir o contexto faltante em vez de adivinhar.</critical>
<critical>Selecionar no maximo um padrao primario. Adicionar um padrao complementar apenas quando ele for indispensavel para fechar o desenho sem duplicar responsabilidade.</critical>
<critical>Executar `python3 scripts/select_pattern.py --input <arquivo-json-ou->` antes de recomendar qualquer pattern.</critical>
<critical>Inicializar o bundle com `python3 scripts/init-bundle.py <slug>` antes de materializar a decisao quando a entrega pedir artefato persistente.</critical>
<critical>Executar `python3 scripts/validate_pattern_bundle.py <bundle_dir>` antes de considerar a saida pronta.</critical>
<critical>Quando a decisao depender de compatibilidade com codebase existente, citar ao menos uma evidencia em formato `path:line`. Se o contexto for greenfield, declarar explicitamente `greenfield`.</critical>
<critical>Todos os artefatos gerados por esta skill DEVEM ficar em PT-BR e sem placeholders.</critical>
<critical>Usar estados finais explicitos: `done`, `needs_input`, `blocked` ou `failed`.</critical>

## Entrada Obrigatoria
- Problema alvo e objetivo tecnico.
- Trecho de codigo, diff, pseudocodigo ou descricao estrutural suficiente para inferir sinais.
- Restricoes reais: desempenho, custo de manutencao, compatibilidade, concorrencia, testabilidade e limite de complexidade.

## Entrada Recomendavel
- Hotspots recorrentes, defeitos observados e historico de mudanca.
- Contratos publicos, invariantes e requisitos de falha.
- Linguagem, paradigma dominante e convencoes do projeto.

## Saida Obrigatoria
- Bundle canonico em `pattern-decisions/<slug>/` quando houver materializacao persistente.
- `bundle.json`, `decision.md`, `implementation.md`, `transcript.md`, `selector-input.json` e `selector-output.json`.
- Resultado do seletor com pattern primario ou decisao explicita de `nao aplicar padrao`.
- Justificativa objetiva de economia, eficiencia e robustez.
- Pseudocodigo canonico e mapeamento por paradigma.
- Plano de implementacao/refatoracao e plano de testes.
- Evidencia de compatibilidade com `path:line` ou declaracao explicita de `greenfield`.

## Contrato de Compatibilidade
- O comportamento deve ser identico entre agentes: a mesma evidencia precisa levar ao mesmo status final e ao mesmo pattern primario.
- `done`: bundle completo, seletor executado, validador executado e sem blockers.
- `needs_input`: falta evidencia material para decidir com seguranca.
- `blocked`: ha impedimento externo ou dependencia ausente fora do controle da skill.
- `failed`: o bundle ficou estruturalmente invalido ou contraditorio.

## Procedimentos

**Etapa 1: Coletar a evidencia minima**
1. Extrair o problema observavel, o comportamento esperado e as restricoes inegociaveis.
2. Localizar a fonte mais forte de verdade nesta ordem:
   - codigo ou diff real
   - pseudocodigo ou tipos/contratos
   - descricao textual detalhada
3. Identificar se o problema esta em criacao, composicao estrutural, distribuicao de comportamento, coordenacao entre objetos, variacao de algoritmo, estado, extensao de operacoes, travessia de estrutura, acesso a recurso, uso de memoria ou acoplamento excessivo.
4. Registrar o que NAO esta provado. Sem essa lista, a skill tende a superprescrever.

**Etapa 2: Normalizar sinais e restricoes**
1. Ler `references/selection-matrix.md` para converter a evidencia em sinais canonicos.
2. Ler `references/efficiency-and-cost-rules.md` para aplicar os gates de economia e cortar overengineering antes da selecao.
3. Se a evidencia estiver ambigua, ler `references/evidence-collection.md` para rodar uma coleta dirigida antes de prosseguir.
4. Montar um JSON com:
   - `problem`
   - `signals`
   - `constraints`
   - `force_reject` quando algum pattern estiver explicitamente proibido pelo contexto
5. Usar apenas sinais canonicos confirmados por evidencia. Nao inventar sinais por similaridade superficial.

**Etapa 3: Rodar o seletor deterministico**
1. Executar `python3 scripts/select_pattern.py --input <arquivo-json-ou->`.
2. Se o retorno vier com `status = needs_more_evidence` ou `status = ambiguous`, parar e solicitar apenas as evidencias faltantes listadas pelo script.
3. Se o retorno vier com `status = reject`, recomendar explicitamente `nao aplicar padrao` e explicar por que a alternativa simples vence em economia, eficiencia e robustez.
4. Se o retorno vier com `status = ok`, registrar:
   - pattern primario
   - padrao complementar, se houver
   - alternativa simples rejeitada
   - patterns rejeitados
   - argumentos de economia, eficiencia e robustez

**Etapa 4: Validar a decisao contra o catalogo**
1. Ler `references/pattern-catalog.md` apenas para o pattern primario e para os patterns rejeitados mais proximos.
2. Confirmar:
   - o problema-alvo bate com a intencao do pattern
   - os sinais fortes estao presentes
   - os sinais de exclusao NAO estao presentes
   - o custo estrutural e aceitavel para o caso
   - o pattern pertence ao conjunto fechado do catalogo classico do Refactoring.Guru
3. Se o pattern escolhido falhar em qualquer um desses pontos, descartar a recomendacao e retornar para a Etapa 2.

**Etapa 5: Produzir o bundle decisorio**
1. Se a entrega for persistente, executar `python3 scripts/init-bundle.py <slug>` para criar `pattern-decisions/<slug>/`.
2. Ler `assets/pattern-decision-template.md`.
3. Preencher todas as secoes obrigatorias com conteudo final, sem placeholders.
4. Em `## Padrao primario`, usar exatamente um destes formatos:
   - `Recomendar: <Pattern Name>`
   - `Recomendar: nao aplicar padrao`
5. Em `## Alternativa mais simples rejeitada`, sempre registrar a solucao direta que quase venceu.
6. Em `## Padroes rejeitados`, listar os patterns proximos e o motivo objetivo da rejeicao, evitando falso positivo futuro.
7. Em `## Pseudocodigo canonico`, descrever a estrutura minima do pattern sem sintaxe dependente de linguagem.
8. Preencher `bundle.json` com `primary_pattern`, `complementary_pattern`, `rejected_patterns`, `status` e `readiness.blockers`.
9. Registrar em `selector-output.json` a saida real do seletor sem reescrever a decisao manualmente.

**Etapa 6: Preparar a implementacao ou refatoracao**
1. Se a tarefa incluir implementacao, ler `references/language-mapping-rules.md`.
2. Ler `assets/pattern-implementation-template.md`.
3. Traduzir o pseudocodigo para o paradigma real do projeto, escolhendo a forma mais simples que preserve a intencao do pattern.
4. Preservar contratos publicos, comportamento externo e invariantes explicitados pelo contexto.
5. Evitar multiplicacao de tipos, heranca ou indirecao desnecessaria. Se a implementacao exigir mais estrutura do que o problema justifica, voltar para `nao aplicar padrao`.

**Etapa 7: Validar a entrega**
1. Ler `references/readiness-gates.md`.
2. Executar `python3 scripts/validate_pattern_bundle.py <bundle_dir>` quando houver bundle persistente; caso contrario, validar ao menos o `decision.md`.
3. Corrigir toda falha apontada pelo `stderr` antes de entregar.
4. Ler `references/validation-checklist.md` e confirmar:
   - ganho de economia real
   - ganho de eficiencia real
   - ganho de robustez real
   - nenhum sinal forte de overengineering
   - plano de testes cobrindo positivo, negativo e regressao
5. Entregar o bundle, o racional da decisao e o plano de implementacao/refatoracao.

## Tratamento de Erros
* Se nao houver codigo, diff, pseudocodigo ou descricao estrutural suficiente, interromper e pedir apenas os dados faltantes em vez de sugerir pattern por analogia.
* Se `scripts/select_pattern.py` retornar `ambiguous`, solicitar o menor conjunto de evidencias adicionais capaz de separar os candidatos em conflito.
* Se `scripts/select_pattern.py` retornar `reject`, nao insistir em aplicar pattern. Explicar por que a opcao simples e tecnicamente superior.
* Se `scripts/validate_pattern_bundle.py` falhar, corrigir o bundle ate retornar `SUCCESS`.
* Se o contexto pedir pattern fora do catalogo classico do Refactoring.Guru, registrar explicitamente que a skill nao cobre esse pattern e responder com `nao aplicar padrao` ou com a aproximacao mais proxima apenas se a equivalencia estiver provada.
