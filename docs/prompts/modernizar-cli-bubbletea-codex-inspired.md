# Prompt Enriquecido: Modernizacao Total da CLI

```text
Objetivo

Modernize a experiencia visual da CLI deste repositorio em Go para um design TUI mais robusto, profissional e operacional, inspirado em Codex, Claude Code e Copilot CLI, sem copiar interfaces literalmente. O foco principal deve ser evoluir a experiencia de execucao do comando task-loop e a camada de apresentacao TUI existente.

Contexto do repositorio

- Linguagem: Go 1.26+
- CLI framework: cobra
- Existe uma superficie real de TUI em:
  - cmd/ai_spec_harness/task_loop.go
  - internal/taskloop/presenter_bubbletea.go
  - internal/taskloop/presenter_bubbletea_test.go
- O projeto ja usa Bubble Tea e Lip Gloss no task-loop.
- Preserve arquitetura, convencoes e fronteiras existentes.
- Prefira a menor mudanca segura com ganho estrutural claro.

Referencia obrigatoria

Use a skill bubbletea como referencia principal de arquitetura e layout. Considere explicitamente:

- layout responsivo baseado em pesos, nao em pixels fixos
- barra de progresso real por lote de tasks
- painel principal com task ativa, fase atual, ferramenta, papel e tempo decorrido
- painel secundario com eventos recentes
- dashboard resumido com contadores: total, pending, in_progress, done, failed, blocked
- truncamento explicito de texto em paineis com borda
- accounting correto de altura/largura considerando bordas
- degradacao elegante para terminais pequenos

Escopo esperado

Proponha uma modernizacao de interface para o task-loop que inclua:

1. Um dashboard superior com identidade visual forte e informacao operacional.
2. Uma barra de progresso horizontal com percentual, contadores e estado atual.
3. Um painel de task ativa mostrando:
   - ID e titulo
   - ferramenta em uso
   - papel atual (executor/reviewer)
   - fase atual
   - tempo decorrido
4. Um painel de fila/resumo mostrando:
   - tasks concluidas
   - tasks em execucao
   - tasks pendentes
   - tasks bloqueadas ou falhadas
5. Um painel de eventos/logs recentes.
6. Um rodape com atalhos de teclado, status da UI e modo efetivo.

Instrucoes de saida

Entregue a resposta em Markdown com as secoes abaixo, nesta ordem:

1. Visao de produto
   - Descreva a experiencia alvo da nova CLI.
   - Explique como ela se aproxima do nivel de robustez percebido em ferramentas como Codex, Claude Code e Copilot CLI.

2. Proposta de arquitetura
   - Explique como adaptar a implementacao atual sem reescrever o dominio.
   - Cite explicitamente os pontos de extensao em:
     - cmd/ai_spec_harness/task_loop.go
     - internal/taskloop/presenter_bubbletea.go
   - Sugira responsabilidades por arquivo se a implementacao for fatiada.

3. Layout visual proposto
   - Mostre um wireframe ASCII completo da tela.
   - Inclua estados para terminal largo e terminal estreito.
   - Mostre claramente onde ficam dashboard, progresso, task ativa, fila e eventos.

4. Referencia no codigo
   - Mostre trechos de codigo Go exemplificando como a estrutura poderia ficar.
   - Inclua exemplos de:
     - struct do model Bubble Tea
     - calculo de layout
     - renderizacao da barra de progresso
     - renderizacao do painel de task ativa
     - truncamento seguro de texto
   - Os exemplos devem ser coerentes com Bubble Tea + Lip Gloss.

5. Plano de implementacao incremental
   - Divida em etapas pequenas e seguras.
   - Em cada etapa, informe impacto, risco e como validar.

6. Criterios de aceitacao
   - Liste criterios objetivos e testaveis.

Restricoes

- Nao invente arquivos ou dependencias sem justificar.
- Nao proponha reescrita completa da CLI se a evolucao incremental for suficiente.
- Nao alterar comportamento publico fora da camada de apresentacao sem explicitar.
- Manter compatibilidade com modo plain quando TUI nao for suportada.
- Comentarios, mensagens e labels devem respeitar o padrao do repositorio em PT-BR.

Criterios de aceitacao obrigatorios

- A proposta deve ancorar a mudanca em arquivos reais deste repositorio.
- Deve haver pelo menos um wireframe ASCII da nova interface.
- Deve haver pelo menos um exemplo realista de codigo Go com Bubble Tea.
- Deve explicar como exibir tasks em execucao e progresso geral do lote.
- Deve considerar responsividade e terminais pequenos.
- Deve preservar separacao entre dominio e apresentacao.

Formato de resposta

- Markdown
- Objetivo e direto
- Sem texto genérico de UX
- Sem depender de bibliotecas fora do ecossistema ja adotado, salvo justificativa concreta
```

## Principais Enriquecimentos

- Ancoragem no codigo real: o prompt referencia [task_loop.go](/Users/jailtonjunior/Git/orchestrator/cmd/ai_spec_harness/task_loop.go:1) e [presenter_bubbletea.go](/Users/jailtonjunior/Git/orchestrator/internal/taskloop/presenter_bubbletea.go:1) para evitar proposta desconectada do repositorio.
- Escopo controlado: em vez de "modernizar a CLI inteira" de forma vaga, o prompt concentra a entrega no `task-loop`, que ja possui base TUI e oferece menor risco.
- Saida deterministica: a resposta pedida agora tem estrutura fixa, com wireframe ASCII, arquitetura, trechos de Go e plano incremental.
- Criterios mensuraveis: o prompt define requisitos verificaveis para reduzir resposta genérica ou superficial.
- Uso correto da referencia Bubble Tea: o texto exige layout responsivo, bordas contabilizadas, truncamento e degradacao elegante, alinhado as regras operacionais da skill.

## Variante Mais Diretiva

```text
Gere uma proposta tecnica para evoluir o task-loop desta CLI Go para uma TUI de nivel profissional inspirada em Codex, Claude Code e Copilot CLI, usando Bubble Tea/Lip Gloss sobre a base existente em cmd/ai_spec_harness/task_loop.go e internal/taskloop/presenter_bubbletea.go. Entregue: wireframe ASCII, arquitetura sugerida, exemplos de codigo Go, barra de progresso, dashboard de tasks, painel de task ativa, painel de eventos, estrategia responsiva e plano incremental de implementacao.
```
