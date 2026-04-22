# Prompt Enriquecido: Task Loop para `prd-unified-version-resolution`

```text
Voce esta executando o fluxo de implementacao do bundle `tasks/prd-unified-version-resolution` neste repositorio usando o binario local compilado.

Objetivo:
- compilar o projeto com `make build`
- usar o comando real `./ai-spec task-loop` para processar as tasks do bundle `tasks/prd-unified-version-resolution`
- repetir a execucao para cada ferramenta suportada neste contexto: `claude`, `copilot`, `gemini` e `codex`
- produzir como resultado apenas alteracoes locais no workspace, visiveis como arquivos modificados ou novos no git
- nao criar commit, nao abrir PR, nao fazer push e nao desfazer manualmente alteracoes validas geradas pelo fluxo

Contexto obrigatorio:
- ler `AGENTS.md` da raiz antes de qualquer acao
- respeitar a governanca e as convencoes locais do repositorio
- considerar que este projeto usa Go, `make build` para compilar e `task-loop` como comando documentado
- usar o bundle alvo exatamente em `tasks/prd-unified-version-resolution`

Ferramentas obrigatorias:
- `claude`
- `copilot`
- `gemini`
- `codex`

Sequencia obrigatoria por ferramenta:
1. Confirmar que o repositorio esta no estado atual do workspace e registrar suposicoes relevantes sem inventar contexto.
2. Executar `make build`.
3. Confirmar que o binario local `./ai-spec` esta disponivel para uso apos o build.
4. Rodar primeiro um `dry-run` do bundle alvo com a ferramenta atual.
5. Se o `dry-run` estiver coerente, executar o `task-loop` real com iteracoes limitadas e `--report-path` explicito.
6. Ao final, inspecionar os arquivos alterados no workspace e o relatorio gerado.
7. Registrar o status da execucao da ferramenta atual como `pass`, `bug`, `blocked` ou `inconclusive`.
8. Repetir o ciclo para a proxima ferramenta sem criar commit entre uma execucao e outra.

Comandos esperados:
- build: `make build`
- dry-run: `./ai-spec task-loop --tool <tool> --dry-run tasks/prd-unified-version-resolution`
- execucao real: `./ai-spec task-loop --tool <tool> --max-iterations 2 --report-path ./docs/reports/task-loop-<tool>-unified-version-resolution.md tasks/prd-unified-version-resolution`

Regras de execucao:
- usar `task-loop`, nao `task loop`
- preservar a menor mudanca segura que resolva a task elegivel
- nao inventar dependencias, versoes ou contexto ausente
- nao marcar sucesso sem evidencias concretas no relatorio, no terminal ou no diff do git
- se uma ferramenta falhar por autenticacao, quota, binario ausente ou limitacao externa, classificar como `blocked` e registrar causa objetiva
- se o `dry-run` indicar ausencia de task elegivel ou bundle inconsistente, registrar isso explicitamente antes de decidir pela execucao real
- se houver alteracoes locais previas nao relacionadas, nao revertelas; apenas registrar o risco de contaminacao da execucao

Saida esperada:
- responder em PT-BR
- usar Markdown
- apresentar uma secao `Resultado`
- apresentar uma secao `Evidencias`
- apresentar uma secao `Arquivos alterados`
- apresentar uma secao `Bugs ou bloqueios`
- para cada ferramenta, listar:
  - comando executado
  - resultado da compilacao
  - resultado do dry-run
  - resultado da execucao real
  - caminho do report
  - resumo do diff gerado no git
  - classificacao final: `pass`, `bug`, `blocked` ou `inconclusive`

Criterios de aceitacao:
- `make build` executado antes do uso do binario
- o bundle alvo usado em todas as execucoes e `tasks/prd-unified-version-resolution`
- cada ferramenta e executada com o comando correto `./ai-spec task-loop --tool <tool> ...`
- o resultado observavel da tarefa deve ser um conjunto de arquivos alterados ou criados no workspace, ainda sem commit no git
- nenhum commit, PR ou push deve ser criado
- divergencias entre ferramentas devem ser registradas explicitamente
- bloqueios externos nao devem ser mascarados como sucesso

Nao fazer:
- nao usar um comando diferente de `task-loop`
- nao pular o build
- nao pular o `dry-run` sem justificativa objetiva
- nao commitar as alteracoes geradas
- nao declarar que a implementacao terminou corretamente sem verificar o diff local no git
- nao esconder erros de autenticacao, quota, permissao ou incompatibilidade da ferramenta
```

## Justificativas das adicoes

- Corrigi a forma do comando para `task-loop` com base na documentacao local do repositorio, evitando que o prompt force um comando inexistente.
- Tornei explicita a matriz de ferramentas `claude`, `copilot`, `gemini` e `codex`, porque "cada ferramenta" estava subespecificado.
- Converti "arquivos uncommit no git" em criterio operacional verificavel: gerar diff local sem commit, PR ou push.
- Estruturei a sequencia `build -> dry-run -> execucao real -> inspecao do diff`, porque isso reduz risco e aumenta rastreabilidade.
- Inclui `--report-path` explicito para cada execucao, permitindo comparar resultados entre ferramentas.
- Adicionei classificacoes finais (`pass`, `bug`, `blocked`, `inconclusive`) para evitar encerramentos vagos.
- Mantive o escopo fechado em `tasks/prd-unified-version-resolution`, sem expandir para outros bundles ou tarefas fora do pedido.
