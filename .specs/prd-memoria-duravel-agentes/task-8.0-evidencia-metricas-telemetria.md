# Tarefa 8.0: Evidência de memória, métricas e telemetria

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar a operação de memória auditável. **Risco alto por consequência, não por tamanho:** mexe com o invariante de posição da seção de métricas no relatório e com gates de evidência baseados em expressão regular — a classe de defeito "gate desligado em silêncio" que o `AGENTS.md` documenta.

Os sete eventos de domínio usam o dispatcher existente, não o enum fechado de `internal/runtime/events/kinds.go`. As métricas entram pelo mapa de campos extra, que já é renderizado genericamente por `internal/runtime/persistence/report.go:71` — nenhum template é tocado.

<requirements>
- RF-34: relatório registra o que foi lido, o que foi gravado, orçamento por camada, compactação, redação e tomada de bastão.
- RF-32: rastrear cada fato até a sessão de origem, pelos metadados do próprio fato.
- RF-35: métricas na telemetria, opt-in via `GOVERNANCE_TELEMETRY`, append-only.
- A seção de evidência é injetada **antes** da seção de métricas: `internal/runtime/persistence/report.go:104` substitui do cabeçalho de métricas até o fim do arquivo, então seção posterior seria apagada.
- Nenhuma expressão regular nova pode usar classe de bracket com caractere multibyte — em processador orientado a byte ela nunca casa e desliga o gate em silêncio.
- Os hooks de memória tratam o próprio erro e o compõem na evidência, em vez de propagá-lo, para não abortar o fan-out sequencial de hooks alheios.
</requirements>

## Subtarefas

- [x] 8.1 Definir os sete eventos de domínio satisfazendo `hooks.Event`: fato registrado, arquivado, promovido, contradição detectada, segredo redigido, compactação executada, bastão transferido.
- [x] 8.2 Definir constantes nomeadas para as chaves de métrica, para que erro de digitação não produza métrica separada.
- [x] 8.3 Emitir as métricas pelo mapa de campos extra de `events.MetricSet`.
- [x] 8.4 Injetar a seção de evidência antes da seção de métricas, com função de injeção seguindo o padrão existente de `report.go`.
- [x] 8.5 Estender `make test-validators` com um relatório contendo a seção nova, provando que os gates de `.agents/scripts/` continuam capturando corretamente.
- [x] 8.6 Adicionar campos de telemetria condicionalmente, seguindo o padrão de `internal/telemetry/acp.go`.
- [x] 8.7 Atualizar `docs/telemetry-feedback-cycle.md` e `docs/troubleshooting.md`.

## Detalhes de Implementação

Ver techspec.md, seção "Monitoramento e Observabilidade". Ver MD-005 (`adr-005-evidencia-metricas-memoria.md`) integralmente — é a ADR desta tarefa.

## Critérios de Sucesso

- O relatório contém todos os seis itens que RF-34 exige.
- **A seção de métricas permanece a última seção** em todos os relatórios gerados, provado por teste.
- `make test-validators` passa com o relatório estendido; nenhum gate de evidência deixa de capturar.
- Nenhuma expressão nova usa bracket com caractere multibyte; o caso é exercitado sob `LC_ALL=C`.
- Cada fato é rastreável até sessão, CLI e data.
- Com `GOVERNANCE_TELEMETRY` desligado, nenhuma métrica é escrita.
- Falha de um hook de memória não impede a execução de hooks registrados depois dele.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/persistence/report.go` — injeção de seção; invariante de posição na linha 104
- `internal/runtime/events/metricset.go` — mapa de campos extra, apenas leitura
- `internal/runtime/hooks/dispatcher.go` — contrato `Event`, apenas leitura
- `internal/telemetry/acp.go` — padrão de campo condicional
- `.agents/scripts/validate-task-evidence.sh` e `validate-review-evidence.sh` — gates a não quebrar
- `tests/scripts/` — teste de validadores a estender
- `docs/telemetry-feedback-cycle.md`, `docs/troubleshooting.md`
