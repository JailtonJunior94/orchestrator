# DOMAIN MODEL

## Titulo
[nome curto do modelo]

## Resumo Executivo
Contexto:
[problema e objetivo em linguagem de negocio]

Decisao central:
[qual fluxo ou comportamento o modelo organiza]

Status de prontidao:
[draft | done | needs_input | blocked | failed]

## Problema e Objetivo
Problema atual:
[descrever o que hoje gera ambiguidade, custo, erro ou retrabalho]

Objetivo de negocio:
[resultado esperado]

Objetivo de modelagem:
[o que precisa ficar inequivoco para destravar implementacao ou decisao]

## Materiais e Evidencias
Materiais usados:
- [arquivo, link, incidente, regra, entrevista ou descricao]

Confronto com codebase:
- Escopo analisado:
- Status do confronto:
- Evidencias:
- Riscos de compatibilidade:

## Escopo e Fora de Escopo
Inclui:
- [item]

Exclui:
- [item]

## Linguagem Ubiqua
| Termo | Definicao | Sinonimos proibidos/ambiguous | Observacoes |
| --- | --- | --- | --- |
| [termo] | [definicao] | [nao usar] | [nota] |

## Bounded Contexts e Fronteiras
Contextos:
- Contexto:
  Objetivo:
  Ownership:
  Fronteiras:

Mapa de contexto:
- [relacao principal entre contextos, upstream/downstream, ACL, conformist ou equivalente]

## Workflow Principal
Gatilho:
[o que inicia o fluxo]

Passos:
1. [passo]
2. [passo]
3. [passo]

Ponto de decisao:
- [decisao relevante]

## Comandos
- Comando:
  Intencao:
  Pre-condicoes:
  Resultado esperado:
  Falhas de negocio:

## Eventos de Dominio
- Evento:
  Quando ocorre:
  Quem observa:
  Impacto:

## Regras, Politicas e Invariantes
Regras de negocio:
- Regra:
  Motivo:

Politicas:
- Politica:
  Entradas:
  Saida/decisao:

Invariantes:
- Invariante:
  Como impedir estado ilegal:

## Estados e Transicoes
Estado inicial:
[estado]

Estados validos:
- [estado]

Transicoes permitidas:
- [origem] -> [destino]: [condicao]

Transicoes proibidas:
- [origem] -> [destino]: [motivo]

## Tipos Conceituais
Entidades:
- [entidade]: [identidade e responsabilidade]

Value Objects:
- [value object]: [restricao ou significado]

Agregados:
- [agregado]: [consistencia, ownership e limite]

## Erros de Dominio
- Erro:
  Quando ocorre:
  Impacto no fluxo:
  Acao esperada:

## Fronteiras Externas e Traducao
Entradas externas:
- [API, fila, arquivo, UI, planilha ou outro]

Saidas externas:
- [evento, notificacao, side effect ou integracao]

Traducao entre dominio e contrato externo:
- [regra de mapeamento, anti-corruption layer, adaptador ou politica de compatibilidade]

## Persistencia, Consistencia e Auditoria
Persistencia necessaria:
- [o que precisa ser persistido e por que]

Consistencia requerida:
- [forte, eventual, hibrida ou nao aplicavel]

Auditoria/rastreabilidade:
- [o que precisa ficar auditavel]

## Observabilidade e Operacao
Sinais minimos:
- Metricas:
- Logs:
- Alertas:

Falhas operacionais relevantes:
- [falha]

Rollback/contingencia:
- [estrategia]

## Economia, Eficiencia e Custos
Decisoes para reduzir custo:
- [simplificacao ou reutilizacao]

Custo cognitivo:
- [o que foi evitado]

Drivers de custo residual:
- [driver]

## Trade-offs e Decisoes
Alternativas rejeitadas:
- [alternativa]: [por que foi rejeitada]

Trade-offs aceitos:
- [trade-off]

Decisoes consolidadas:
- [decisao]

## Itens em Aberto
- [item pendente ou "Nenhum item aberto bloqueante."]

## Proximo Passo Recomendado
[technical-discovery-production | epic-story-discovery | implementacao guiada] com [objetivo do handoff]
