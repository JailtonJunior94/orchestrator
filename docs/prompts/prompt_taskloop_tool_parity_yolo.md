# Prompt: Paridade Mandatoria do internal/taskloop com Modo Yolo

Atue como engenheiro senior do `internal/taskloop` deste repositorio. Trabalhe de forma mandataria, economica e verificavel.

Objetivo: alinhar a execucao do `internal/taskloop` para que `Claude Code`, `Codex`, `Gemini CLI` e `Copilot CLI` operem com o mesmo comportamento efetivo de autonomia total, usando a flag nativa equivalente a `modo yolo` em cada ferramenta.

Considere como comandos confirmados:

- `codex --yolo`
- `gemini --yolo`
- `copilot --yolo`
- `claude --dangerously-skip-permissions`

Regras obrigatorias:

1. Trate `--dangerously-skip-permissions` no Claude como equivalente funcional ao `--yolo` das demais ferramentas.
2. Nao exija identidade literal de flags entre ferramentas; exija paridade comportamental.
3. Inspecione o estado real do repositorio antes de propor mudancas.
4. Preserve a arquitetura atual e aplique a menor mudanca segura.
5. Nao invente comandos, wrappers, adapters, aliases ou variaveis sem evidência no codigo.
6. Se houver divergencia real entre invocadores, aponte o arquivo e a mudanca exata necessaria.
7. Se ja houver paridade suficiente, declare isso explicitamente e informe o que falta apenas para endurecer a garantia.

Escopo minimo de inspecao:

- `internal/taskloop/`
- `cmd/ai_spec_harness/task_loop.go`
- `AGENTS.md`
- `CLAUDE.md`
- `CODEX.md`
- `GEMINI.md`

Criterios de aceite:

- Existe uma definicao explicita de paridade para as quatro ferramentas.
- `internal/taskloop` passa a invocar cada CLI com sua flag de autonomia total equivalente.
- Claude usa `--dangerously-skip-permissions` como equivalente funcional de `--yolo`.
- Codex, Gemini e Copilot usam `--yolo` quando suportado pela CLI real adotada no repositorio.
- O resultado informa se a mudanca foi em codigo, documentacao ou ambos.
- O resultado lista arquivos alterados e comando de validacao executado.

Contrato de saida:

- Formato: Markdown.
- Seja curto e assertivo.
- Entregue nesta ordem:
  1. `Diagnostico`
  2. `Decisao`
  3. `Mudancas necessarias`
  4. `Arquivos afetados`
  5. `Validacao`
  6. `Riscos ou gaps`

Nao faca:

- Nao responda com teoria geral sobre agentes.
- Nao normalize diferencas reais entre ferramentas sem verificar.
- Nao trate `gh copilot suggest -t shell` como equivalente a `copilot --yolo` sem validar no contexto do repositorio.
- Nao altere comportamento publico sem explicitar.

Tratamento de falhas:

- Se alguma CLI nao suportar a flag esperada neste ambiente, registre a evidencia e proponha a alternativa funcional equivalente.
- Se o repositorio estiver inconsistente entre codigo e documentacao, priorize o codigo executavel e aponte a documentacao a corrigir.

## Justificativas das Adicoes

- Explicitei que a equivalencia desejada e comportamental, porque o exemplo do Claude contradiz a exigencia de usar literalmente `--yolo`.
- Limitei o escopo aos arquivos mais relevantes para reduzir resposta dispersa.
- Adicionei criterios de aceite verificaveis para forcar uma resposta objetiva.
- Restringi a saida em Markdown curto para manter o prompt enxuto.
