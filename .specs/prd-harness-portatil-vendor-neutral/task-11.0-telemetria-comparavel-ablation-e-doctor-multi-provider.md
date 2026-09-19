# Tarefa 11.0: Telemetria comparavel, ablation e doctor multi-provider

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar as metricas novas efetivamente observaveis e agregar o diagnostico em blocos por provedor.

<requirements>
- RF-30
- RF-31
- RF-32
- RF-33
- RF-34
- RF-43
- RF-43.1
- RF-44
- RF-45
- RF-46
- RF-47
</requirements>

## Subtarefas

- [ ] 11.1 Estender a ESCRITA e o LEITOR: o parser hoje reconhece apenas duas chaves e descarta o resto sem default (V-29)
- [ ] 11.2 Round-trip por campo lendo via o parser de PRODUCAO, nunca por inspecao do arquivo
- [ ] 11.3 Metrica ausente e AUSENTE, nunca zero, nunca inferida
- [ ] 11.4 Tabela de campos sensiveis que falha se qualquer um for emitido
- [ ] 11.5 Mecanismo e baseline de ablation; campanhas ficam sob demanda e nao sao gate
- [ ] 11.6 V-37: assertar que toda metrica usada tem escritor de producao identificado — o mapa de fluxos e sempre vazio hoje
- [ ] 11.7 Doctor com blocos Core e um por provedor, atribuicao de falha TIPADA e exit != 0
- [ ] 11.8 Doctor consome o resultado da verificacao de integridade entregue na tarefa 1.0; nao a reimplementa
- [ ] 11.9 `verify` preserva integralmente contrato, flags, saida e codigos de saida

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Chave nova aparece em relatorio, resumo e tendencia — presenca no arquivo de log nao basta
- Nenhum leitor existente quebra e chave desconhecida continua ignorada em vez de falhar
- Zero metrica inventada, inferida ou preenchida com zero
- Doctor reporta os quatro provedores, com somente verificacoes estaticas e sem chamada paga a LLM

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
