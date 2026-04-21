# Prompt: Execução Sequencial Automatizada de Tasks por Agente de IA

## Contexto do Repositório

Este repositório usa Go 1.26+, cobra CLI, skill `execute-task` como procedimento canônico de implementação,
e suporta os agentes: Claude Code, Codex, Gemini CLI e GitHub Copilot.

A estrutura padrão de um PRD é:

```
tasks/prd-<feature-slug>/
├── prd.md
├── techspec.md
├── tasks.md
├── 1_task.md
├── 1_task_review.md   ← excluir da descoberta
├── 2_task.md
└── ...
```

Problema que este prompt resolve: hoje não há mecanismo automatizado para executar tasks sequencialmente
sem intervenção manual a cada iteração.

---

## Objetivo

Você é um agente orquestrador responsável por executar todas as tasks elegíveis de um PRD de forma sequencial,
criando uma nova sessão de agente isolada para cada task, capturando o resultado e produzindo um relatório final.

---

## Pré-condições (validar antes de qualquer execução)

1. O argumento `$1` (caminho da pasta PRD) foi fornecido e é um caminho absoluto ou relativo válido.
2. A pasta existe e contém exatamente os três artefatos obrigatórios: `tasks.md`, `prd.md`, `techspec.md`.
3. Existe ao menos um arquivo `*_task.md` na pasta (excluindo `*_task_review.md`).

Se qualquer pré-condição falhar, interromper imediatamente com mensagem de erro estruturada:

```
ERRO: <motivo>
Caminho fornecido: <$1>
Arquivos encontrados: <lista ou "nenhum">
```

---

## Algoritmo de Execução

### Passo 1 — Descobrir e ordenar tasks

- Listar todos os arquivos com padrão `*_task.md` na pasta `$1`.
- Excluir arquivos que correspondam a `*_task_review.md`.
- Extrair o número prefixo de cada arquivo (ex: `1_task.md` → `1`, `10_task.md` → `10`).
- Ordenar numericamente de forma crescente (não lexicográfica).

### Passo 2 — Para cada task (em ordem)

**a. Verificar se já está completa**

- Ler `tasks.md` e localizar a linha correspondente à task pelo número.
- Se a linha contiver `[x]` (checkbox marcado), classificar como `PULADA` e avançar para a próxima.

**b. Detectar skills necessárias**

Ler o arquivo `<N>_task.md` e identificar tecnologias envolvidas:

| Sinal encontrado no task file | Skill a carregar |
|---|---|
| `.go`, `Go`, `golang` | `.agents/skills/go-implementation/SKILL.md` |
| `.ts`, `.js`, `Node`, `TypeScript` | `.agents/skills/node-implementation/SKILL.md` |
| `.py`, `Python` | `.agents/skills/python-implementation/SKILL.md` |
| bug, correção, regression | `.agents/skills/bugfix/SKILL.md` |
| refactor, refatoração | `.agents/skills/refactor/SKILL.md` |

Sempre incluir como base obrigatória:
- `AGENTS.md`
- `.agents/skills/agent-governance/SKILL.md`
- `.agents/skills/execute-task/SKILL.md`

**c. Compor o prompt da task**

```
Você é um assistente IA responsável por implementar tarefas de forma correta e verificável.

Ative e siga a skill execute-task para conduzir todo o processo de implementação.
A skill contém o procedimento completo: validação de elegibilidade, carregamento de contexto,
implementação, validação, revisão e persistência de evidências.

Carregue também as seguintes skills adicionais identificadas para esta tarefa:
<lista das skills detectadas no passo b>

Utilize o Context7 MCP para consultar documentação de linguagens, frameworks e bibliotecas
envolvidas na implementação antes de editar código.

VOCÊ DEVE iniciar a implementação logo após o planejamento, sem aguardar confirmação do usuário.

Após completar a tarefa com sucesso:
1. Marque a tarefa como completa em tasks.md (checkbox [x]).
2. Execute o task-reviewer conforme exigido pela skill execute-task.

Implemente a tarefa ${task_num} do PRD localizado em ${PRD_DIR}.

Artefatos de referência:
- Task:     ${PRD_DIR}/${task_num}_task.md
- PRD:      ${PRD_DIR}/prd.md
- Tech Spec: ${PRD_DIR}/techspec.md
- Tasks:    ${PRD_DIR}/tasks.md
```

**d. Invocar o agente em nova sessão isolada**

Criar um subprocess isolado (nova sessão, sem herdar estado da sessão atual) com o agente escolhido.

Mapeamento de flags de autonomia total por ferramenta:

| Ferramenta | Comando | Flag de autonomia |
|---|---|---|
| Claude Code | `claude` | `--dangerously-skip-permissions` |
| Codex | `codex` | `--yolo` |
| Gemini CLI | `gemini` | `--yolo` |
| GitHub Copilot | `copilot` | `--yolo` |

Exemplo de invocação para Claude:

```bash
claude --dangerously-skip-permissions -p "<prompt composto no passo c>"
```

A ferramenta deve ser selecionada antes da execução (argumento `$2` ou variável de ambiente `TASK_AGENT`).
Se não definida, usar `claude` como padrão.

**e. Capturar exit code e decidir**

- Exit code `0`: classificar task como `EXECUTADA`.
- Exit code diferente de `0`: classificar como `FALHA`, logar o erro com task number e exit code.
  - Se `--non-interactive`: parar a execução e exibir o relatório parcial.
  - Se `--interactive` (padrão): perguntar ao usuário:
    ```
    Task <N> falhou (exit code <X>). Continuar para a próxima? [s/N]:
    ```
  - Continuar somente com confirmação explícita `s` ou `S`.

**f. Garantir isolamento de sessão**

Cada agente deve ser invocado como processo independente, sem compartilhar contexto de sessão com tasks anteriores.
Não reutilizar processos ou sessões entre tasks.

---

## Passo 3 — Relatório Final

Ao concluir todas as tasks (ou após interrupção por falha em modo não-interativo), exibir:

```markdown
## Relatório de Execução de Tasks

PRD: <caminho>
Agente: <ferramenta usada>
Data: <timestamp ISO 8601>

| Task | Arquivo              | Status    | Exit Code | Observação          |
|------|----------------------|-----------|-----------|---------------------|
| 1    | 1_task.md            | EXECUTADA | 0         |                     |
| 2    | 2_task.md            | PULADA    | -         | já marcada como [x] |
| 3    | 3_task.md            | FALHA     | 1         | erro: <mensagem>    |

Resumo: <N> executadas · <N> puladas · <N> falhas
```

---

## Critérios de Aceitação

- [ ] Pré-condições validadas antes de qualquer execução; falha imediata com mensagem estruturada se inválidas.
- [ ] Tasks ordenadas numericamente (não lexicograficamente).
- [ ] Tasks marcadas com `[x]` em `tasks.md` são puladas sem invocação do agente.
- [ ] Skills detectadas dinamicamente com base no conteúdo do task file.
- [ ] Cada task executada em processo isolado (nova sessão).
- [ ] Exit code capturado e tratado: `0` = sucesso, qualquer outro = falha.
- [ ] Comportamento interativo vs. não-interativo controlável.
- [ ] Relatório final emitido sempre, mesmo em caso de interrupção parcial.
- [ ] Ferramenta de agente configurável sem alterar o prompt.

---

## Restrições

- Não compartilhar estado de sessão entre tasks.
- Não pular tasks com falha silenciosamente.
- Não executar tasks fora da ordem numérica definida.
- Não modificar `tasks.md` diretamente — a atualização do checkbox é responsabilidade da skill `execute-task` dentro de cada sessão.
- Não inventar flags de CLI sem verificar suporte real no ambiente.

---

## Justificativas das Adições

- **Contexto do repositório**: necessário para que o agente saiba a estrutura esperada e não infira padrões de outro projeto.
- **Critérios de aceitação mensuráveis**: o prompt original não tinha condição de done verificável — adicionei checkboxes que mapeiam diretamente para comportamento observável.
- **Detecção de skills por sinal no task file**: resolve a ambiguidade "verificar quais skills precisam ser carregadas" com heurística determinística e tabela explícita.
- **Tabela de flags por ferramenta**: resolve "funcionar de forma igualitária" com paridade comportamental documentada, em vez de paridade literal de flags.
- **Modo interativo vs. não-interativo**: elimina a ambiguidade "perguntar se continua ou para" — comportamento padrão é interativo, mas pode ser desligado com flag.
- **Isolamento de sessão definido como subprocess independente**: remove a ambiguidade "nova instância/sessão".
- **Relatório estruturado em tabela**: o original não definia formato de saída — adicionei contrato explícito com colunas verificáveis.
- **Ferramenta configurável via `$2` ou variável de ambiente**: permite reutilizar o mesmo prompt para todos os agentes sem edição.
