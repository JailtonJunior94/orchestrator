# Prompt enriquecido: execução e validação do task-loop (financialcontrol-api)

## Objetivo

Executar e validar o ciclo completo do `task-loop` no projeto `financialcontrol-api`, garantindo que a implementação siga os padrões de rigor, captura de evidências e paridade estabelecidos no `orchestrator`, utilizando modelos específicos para cada fase do processo.

## Contexto do Projeto

- **Caminho Alvo:** `/Users/jailtonjunior/Git/financialcontrol-api/tasks/prd-modernization-database-devkit-go`
- **Stack:** Go, Makefile, `ai-spec-harness`.
- **Referências de Rigor (Orchestrator):**
  - `tasks/prd-taskloop-execution-validation`: Ciclo Selector -> Execution -> Acceptance Gate -> Evidence Recorder -> Final Review -> Bugfix Loop.
  - `tasks/prd-portability-parity`: Evidências "hardened", telemetria e paridade.

## Restrições de Modelos (MANDATÓRIO)

| Fase | Modelo Alvo |
| :--- | :--- |
| **Instalação, Upgrade, Doctor e Execute Task** | `claude` (Sonnet 4.6) |
| **Review Final e Validação de Diff** | `opus 4.7` |

## Prompt Enriquecido para o Agente

```text
Você deve atuar como um engenheiro de software sênior operando o `task-loop` para o projeto `financialcontrol-api`.

### 1. Preparação, Modelos e Contexto
- **Restrição de Modelo:** Todas as etapas de execução devem usar Sonnet 4.6. A etapa de Review deve usar Opus 4.7.
- Mude o diretório de trabalho para: `/Users/jailtonjunior/Git/financialcontrol-api`
- Antes de iniciar o bundle de tasks, valide o ambiente:
  1. Execute o fluxo de **Instalação** (`install`) do harness.
  2. Execute o fluxo de **Upgrade** (`upgrade`) para garantir a última versão das skills.
  3. Execute o fluxo de **Doctor** (`doctor`) para validar a saúde do ambiente.
- Localize o bundle de tasks em: `tasks/prd-modernization-database-devkit-go`
- Leia: `AGENTS.md`, `.agents/skills/agent-governance/SKILL.md` e o `tasks.md` do bundle.

### 2. Execução do Ciclo Task-Loop (Sonnet 4.6)
Para a próxima task pendente:
1. **Seleção:** Identifique a tarefa em `tasks.md`.
2. **Implementação:** Use a skill `execute-task` para realizar as mudanças.
3. **Build:** Execute `make build` no diretório raiz.
4. **Validação de Caso Real:**
   - Execute testes unitários/integração.
   - Aplique o rigor de `tasks/prd-taskloop-execution-validation`: trate fluxos de erro e isole a lógica de domínio.

### 3. Registro de Evidências (Hardening)
- Capture o output do `make build`, `install`, `upgrade`, `doctor` e dos testes.
- Registre o diff gerado e adicione a seção "Evidência de Validação" no arquivo da task.
- Siga o padrão de `tasks/prd-portability-parity`.

### 4. Revisão (Opus 4.7)
- **Mandatário:** Troque para o modelo **Opus 4.7** para realizar esta etapa.
- Execute a skill `review` sobre o diff consolidado.
- Avalie se a paridade e o rigor foram mantidos.
- Se houver achados críticos, retorne para o Sonnet 4.6 para o `bugfix` (máximo 3 iterações).

### 5. Fechamento
- Atualize o status da task em `tasks.md` para `done`.

### Definição de Pronto (DoP)
- [ ] Fluxos de Instalação, Upgrade e Doctor validados com sucesso.
- [ ] Task concluída com evidências (logs de build e testes) no arquivo .md.
- [ ] Review realizada pelo Opus 4.7 sem achados críticos pendentes.
- [ ] `tasks.md` atualizado com o progresso real.
```

## Justificativa das Adições (Prompt-Enricher)

| Adição | Justificativa |
|--------|---------------|
| **Seleção de Modelos** | Atende à exigência mandatória de usar Sonnet 4.6 para execução e Opus 4.7 para crítica/revisão. |
| **Ciclo de Ciclo Completo** | Inclui explicitamente Instalação, Upgrade e Doctor como pré-requisitos de validação. |
| **Caminho Absoluto** | Garante consistência em múltiplos workspaces. |
| **Rigor de Referência** | Vincula a execução aos padrões de paridade e evidências do `orchestrator`. |
| **Definição de Pronto** | Critérios claros para garantir que todas as fases (incluindo a troca de modelo) foram cumpridas. |
