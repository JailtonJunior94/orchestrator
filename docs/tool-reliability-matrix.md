# Matriz de Confiabilidade por Ferramenta

> Referencia operacional para escolher Claude, Codex, Gemini ou Copilot com menos subjetividade.
>
> Este documento resume papel recomendado, risco operacional e custo de contexto com base em evidencias reais ja registradas no repositorio. Ele complementa o [Playbook Mestre de Desenvolvimento](development-playbook.md), o [Scorecard de Qualidade e Confianca](quality-scorecard.md) e a [Matriz de degradacao](degradation-matrix.md).

## Objetivo

Dar um criterio curto para decidir:

- qual ferramenta tende a funcionar melhor como executor
- qual ferramenta exige mais supervisao
- quando o custo de contexto ou o risco operacional sobe demais
- quando vale separar papel de executor e reviewer

## Base Primaria

Esta matriz foi consolidada a partir de fontes reais deste repositorio:

- [Guia de troubleshooting](troubleshooting.md)
- [Matriz de degradacao](degradation-matrix.md)
- [ADR-007 — Copilot CLI stateless workaround](adr/007-copilot-cli-stateless-workaround.md)
- `docs/bug_report_claude_auth_subprocess.md`
- `docs/bug_report_codex_maturidade.md`
- `docs/bug_report_gemini_quota_throttling.md`
- `docs/bug_report_gemini_tasks_gitignore.md`

## Como Ler a Matriz

- `Papel recomendado`: onde a ferramenta mostrou melhor relacao entre confianca e custo.
- `Forca principal`: onde os relatos reais mostram vantagem objetiva.
- `Risco operacional`: o modo de falha mais importante observado no repositorio.
- `Custo e contexto`: quanto esforco extra a ferramenta tende a exigir para operar bem.

## Matriz

| Ferramenta | Papel recomendado | Forca principal | Risco operacional | Custo e contexto | Uso pratico recomendado |
| --- | --- | --- | --- | --- | --- |
| Claude | reviewer ou executor assistido em sessao interativa ja autenticada | Boa aderencia a governanca local quando roda na sessao principal; docs e hooks especificos da tool ajudam a manter trilho | Em subprocesso via harness, falha de autenticacao bloqueia o fluxo (`Not logged in · Please run /login`) | Medio para uso manual; alto para automacao nao interativa se `ANTHROPIC_API_KEY` ou sessao valida nao estiverem disponiveis | Use para revisao e execucao unitario-interativa. Evite como executor de `task-loop` ou self-testing sem preflight explicito de auth. |
| Codex | executor principal para task unitaria ou lote pequeno supervisionado | Foi a unica ferramenta com `pass` explicito na execucao real do bundle `maturidade`; usa `AGENTS.md` diretamente e nao depende de wrapper documental exclusivo | Em alguns cenarios de subprocesso houve zero output e baixa observabilidade, o que dificulta diagnostico quando trava ou expira | Baixo a medio em execucao direta; medio quando a tarefa exige acompanhar progresso fino ou quando a CLI roda via pipe | Use como default para implementar uma task por vez. Para review, funciona bem quando o diff e pequeno e a evidencia de validacao ja existe. |
| Gemini | executor secundario para tarefas longas ou exploratorias com timeout realista | Demonstrou atividade real e leitura de contexto; tende a trabalhar de forma substantiva quando recebe tempo suficiente | Pode sofrer com timeout, quota e degradacao de leitura quando `.specs/` fica escondido por ignore; tambem emitiu aviso de MCP em validacoes reais | Medio a alto: pede timeout maior, leitura de output mais paciente e verificacao extra de acesso aos arquivos | Use quando voce aceita throughput menor em troca de exploracao. Nao trate timeout curto como sinal automatico de incapacidade. |
| Copilot | reviewer assistido ou executor com supervisao forte e contexto reinjetado | Produziu o output mais detalhado nos relatos, leu governanca obrigatoria e executou validacoes de forma observavel | O CLI e stateless; sem injecao de contexto tende a perder governanca. Em execucao real ja interpretou `spec-drift` de forma conservadora e bloqueou task | Alto no CLI, porque cada invocacao precisa de contexto explicitamente reinjetado; menor no chat integrado que respeita `copilot-instructions.md` | Use para revisao assistida, triagem e validacao observavel. Como executor, prefira cenarios com wrapper do harness ou prompt rigoroso e contexto fechado. |

## Escolha Rapida por Papel

| Cenario | Ferramenta preferida | Motivo |
| --- | --- | --- |
| Executar uma task isolada com menor subjetividade | Codex | Melhor resultado observado em execucao real e baixo atrito para seguir `AGENTS.md`. |
| Revisar uma task ja implementada com foco em governanca e checklist | Claude ou Copilot | Claude e forte em sessao interativa; Copilot deixa rastro detalhado quando o contexto e reinjetado corretamente. |
| Explorar uma task longa, ambigua ou com leitura extensa | Gemini | Mostrou trabalho real sob timeout maior, mas precisa de supervisao operacional maior. |
| Rodar lote pequeno automatizado | Codex, com supervisao | E a opcao com melhor equilibrio entre execucao real bem-sucedida e menor dependencia de workarounds tool-specific. |
| Rodar subprocesso sem margem para falha de autenticacao | Evitar Claude | O risco observado no repositorio e bloqueante, nao apenas teorico. |

## Observacoes Praticas de Uso

### Claude

- Trate autenticacao como preflight obrigatorio quando houver subprocesso ou harness.
- Se a sessao nao estiver autenticada de forma utilizavel pelo subprocesso, o fluxo para antes da task com erro claro.
- Quando a execucao e manual e interativa, o risco cai bastante; o problema registrado e operacional, nao de qualidade de analise.

### Codex

- E a referencia mais segura para `execute-task` neste repositorio hoje.
- O ponto fraco observado nao foi qualidade de decisao, mas opacidade em alguns modos de execucao por pipe.
- Se a sessao precisar de telemetria humana de progresso, combine Codex com validacoes curtas e checkpoints frequentes.

### Gemini

- Timeout curto de teste distorce a percepcao de confiabilidade.
- Antes de culpar a ferramenta, confirme quota, acesso real a `.specs/` e mensagens de MCP.
- Use quando houver tempo para a ferramenta ler bastante contexto e devolver trabalho substantivo.

### Copilot

- Diferencie sempre Copilot CLI de Copilot Chat no editor.
- No CLI, considere contexto nao reinjetado como risco estrutural, nao erro eventual.
- O valor principal da ferramenta, pelos relatos atuais, e produzir trilha observavel de leitura e validacao.

## Heuristicas Operacionais

1. Se voce precisa de um executor default para `execute-task`, comece por Codex.
2. Se a revisao precisa ser rastreavel e explicita, Claude interativo ou Copilot com contexto reinjetado tendem a ser opcoes melhores que uma execucao cega.
3. Se a ferramenta falhou por autenticacao, timeout curto ou ignore incorreto, classifique isso como risco operacional do ambiente antes de concluir que o modelo nao serve.
4. Se o objetivo for automacao previsivel, prefira ferramentas com menor dependencia de estado externo oculto.

## Limites Deliberados

- Esta matriz nao redefine contratos das skills.
- Esta matriz nao substitui evidencias de execucao da task atual.
- Novos relatos cross-CLI podem mudar a recomendacao; quando isso ocorrer, este documento deve ser atualizado junto com a evidencia primaria correspondente.
