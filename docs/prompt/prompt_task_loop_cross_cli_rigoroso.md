# Prompt para Task Loop Cross-CLI

```text
Voce esta executando uma validacao end-to-end do `task-loop` deste repositorio por meio do binario local `./ai-spec`.

Objetivo:
- compilar a versao atual do projeto
- executar o `task-loop` com as CLIs `claude`, `copilot`, `gemini` e `codex`
- verificar se o mesmo prompt operacional e compreendido e executado corretamente por todas as ferramentas
- usar como suites de validacao os bundles:
  - `tasks/maturidade`
  - `tasks/prd-interactive-task-loop`
  - `tasks/prd-task-loop-modelos-por-papel`
  - `tasks/prd-task-loop-sequential-execution`
  - `tasks/prd-telemetry-feedback`
- quando identificar falha, incompatibilidade, ambiguidade de prompt, divergencia de comportamento, ou erro estrutural do harness, gerar um relatorio de bug em Markdown com evidencia objetiva

Modo de execucao:
1. Ler `AGENTS.md` na raiz do repositorio antes de qualquer acao.
2. Preservar a menor mudanca segura; nao reescrever fluxo, contrato ou arquitetura sem necessidade comprovada.
3. Trabalhar sempre a partir do estado real do repositorio; nao inventar contexto ausente.
4. Antes de executar os loops, compilar a versao atual do binario local.
5. Usar os comandos reais de `task-loop` suportados por cada ferramenta, respeitando as flags de autonomia esperadas pelo projeto.
6. Executar primeiro um `dry-run` por bundle e por ferramenta sempre que isso reduzir risco e esclarecer elegibilidade.
7. Se o dry-run estiver coerente, executar o loop real em lote pequeno, com rastreabilidade.
8. Registrar diferencas observaveis entre ferramentas: parsing do prompt, respeito a `AGENTS.md`, atualizacao de status, geracao de relatorio, comportamento de streaming, review e fallback.
9. Se a falha for do prompt ou da compatibilidade da CLI com o harness, nao mascarar o problema com workarounds silenciosos.

Bundles obrigatorios de validacao:
- `tasks/prd-unified-version-resolution`

Matriz obrigatoria:
- ferramenta: `claude`
- ferramenta: `copilot`
- ferramenta: `gemini`
- ferramenta: `codex`

Comandos-alvo esperados:
- Claude: `./ai-spec task-loop --tool claude <prd-folder>`
- Copilot: `./ai-spec task-loop --tool copilot <prd-folder>`
- Gemini: `./ai-spec task-loop --tool gemini <prd-folder>`
- Codex: `./ai-spec task-loop --tool codex <prd-folder>`

Sequencia minima por ferramenta e bundle:
1. Compilar o projeto.
2. Rodar dry-run do `task-loop`.
3. Rodar execucao real com `--max-iterations` baixo e `--report-path` explicito.
4. Inspecionar resultado, status das tasks e relatorio gerado.
5. Classificar o resultado em `pass`, `bug`, `blocked` ou `inconclusive`.

Criterios de aceitacao:
- o prompt deve ser seguido sem depender de interpretacao vaga
- a ferramenta deve executar o `task-loop` correto para o bundle alvo
- o agente deve respeitar o contrato do repositorio, incluindo leitura de `AGENTS.md`
- a execucao deve produzir evidencia observavel em terminal e/ou em relatorio
- falhas devem ser descritas com causa provavel e reproduzibilidade
- nao declarar sucesso sem evidencias concretas

Quando houver bug:
- gerar um relatorio Markdown enxuto e objetivo
- incluir:
  - titulo
  - data
  - ferramenta
  - bundle
  - comando executado
  - comportamento esperado
  - comportamento observado
  - evidencia objetiva
  - impacto
  - hipotese de causa raiz
  - severidade
  - proxima acao recomendada
- nome sugerido do artefato: `bug_report_<tool>_<bundle>.md`

Regras de saida:
- responder em PT-BR
- ser rigoroso, enxuto e assertivo
- priorizar fatos observados, nao opinioes
- usar Markdown
- separar claramente `Resultado`, `Evidencias` e `Bugs`
- se nao houver bug, declarar explicitamente `Sem bug reproduzido`
- se houver bloqueio externo, declarar explicitamente `Blocked` com a razao

Nao fazer:
- nao supor que todas as CLIs se comportam igual sem verificar
- nao pular compilacao
- nao pular dry-run quando houver risco de executar bundle incorreto
- nao marcar `pass` se o fluxo nao gerou evidencia suficiente
- nao esconder divergencias entre ferramentas
```

## Justificativas das adicoes

- Tornei explicita a sequencia operacional `compilar -> dry-run -> execucao real -> inspecao`, porque o objetivo e validar comportamento do harness, nao apenas produzir output.
- Converti os exemplos de `tasks/` em bundles obrigatorios de validacao para reduzir ambiguidade sobre o que deve ser exercitado.
- Fixei a matriz por ferramenta para garantir paridade observavel entre `claude`, `copilot`, `gemini` e `codex`.
- Inclui criterios de aceitacao mensuraveis para evitar respostas vagas ou sucesso sem evidencia.
- Estruturei o bug report com campos minimos reutilizaveis para comparacao entre CLIs e reproducao posterior.
- Mantive o texto enxuto e prescritivo para maximizar aderencia em execucao nao interativa via `task-loop`.
