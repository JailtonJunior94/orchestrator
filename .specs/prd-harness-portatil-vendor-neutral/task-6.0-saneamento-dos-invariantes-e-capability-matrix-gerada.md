# Tarefa 6.0: Saneamento dos invariantes e capability matrix gerada

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Sanear a fonte antes de publica-la e emitir a matriz versionada com seus dois gates.

<requirements>
- RF-15
- RF-17
- RF-18
- RF-18.1
- RF-19
- RF-20
- RF-21
</requirements>

## Subtarefas

- [ ] 6.1 RF-18.1 PRIMEIRO: escopo passa a derivar de `AppliesTo` e nunca de `Level` (V-26)
- [ ] 6.2 Marcar como evidencia INVALIDA os invariantes satisfeitos pelos stubs que o proprio gerador injeta (V-25)
- [ ] 6.3 Gerador com dois emissores na MESMA execucao: JSON, que o gate compara, e Markdown, que o mantenedor le
- [ ] 6.4 Estender `ValidateParityMatrix` preservando a literal `no dispatch proof test associated`, em vez de criar validador paralelo
- [ ] 6.5 Estados honestos: suportado com teste resolvido, nao suportado, capability de provedor e desconhecido
- [ ] 6.6 Gate do gate cobrindo matriz vazia, drift so no JSON, drift so no Markdown e celula suportada sem teste
- [ ] 6.7 REGISTRO ANTI-GATE-ORFAO nos quatro pontos

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Contagem de celulas suportadas sem teste resolvido e ZERO
- Celulas desconhecidas sao rastreadas e justificadas uma a uma
- Adulterar qualquer um dos dois artefatos derruba o CI
- Reajuste de largura de coluna no Markdown regenerado NAO produz falso positivo
- Teste amarra o conjunto de stubs injetados ao conjunto marcado como evidencia invalida

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Ver `techspec.md`, seção *Arquivos Relevantes e Dependentes*.
