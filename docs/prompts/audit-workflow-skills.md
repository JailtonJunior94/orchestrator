# Auditoria de Fluxo de Trabalho (Core Skills)

## Objetivo
Analisar rigorosamente a integração, conformidade e paridade das skills centrais do `orchestrator`: `create-prd`, `create-technical-specification`, `create-tasks`, `execute-task` e `execute-all-tasks`. O foco é garantir que o ecossistema funcione de forma determinística, segura e eficaz em qualquer projeto, com carregamento sob demanda e rastreabilidade total.

## Contexto Mandatório
- **Stack:** Go 1.26+, spf13/cobra.
- **Workflow:** Sequencial (PRD -> TechSpec -> Tasks -> Execution).
- **Integração:** Baseada em hashes (`spec-hash-prd`, `spec-hash-techspec`) e arquivos de marcação.
- **Paridade:** Deve suportar Claude Code, Copilot CLI, Codex e Gemini CLI através de variáveis padronizadas (`AI_TASKS_ROOT`, `AI_PRD_PREFIX`) e ganchos (`hooks/`).
- **Universalidade:** O sistema deve ser agnóstico ao projeto. A lógica core NÃO deve conter caminhos ou regras fixas que funcionem apenas no repositório `orchestrator`.
- **Isolamento e Contexto (execute-all-tasks):** É extremamente mandatório e obrigatório que o `execute-all-tasks` carregue a skill específica de cada tarefa individualmente. Esta marcação de skill deve ocorrer logo após os critérios de aceite. Ao concluir uma tarefa, é obrigatório limpar o contexto e começar a próxima do zero em todas as ferramentas.
- **Execução de Tarefas (execute-task):** É extremamente mandatório e obrigatório carregar a skill de cada task, marcar rigorosamente após os critérios de aceite e garantir que a tarefa realmente terminou com **zero falso positivo** antes de passar para o próximo estado.

## Itens de Verificação (Análise Minuciosa)

### 1. Rastreabilidade e Drift (Linked Nature)
- [ ] **PRD -> TechSpec:** A techspec injeta o hash do PRD (`<!-- spec-hash-prd: ... -->`)?
- [ ] **TechSpec -> Tasks:** O `tasks.md` injeta os hashes do PRD e da TechSpec?
- [ ] **Tasks -> Execution:** O `execute-task` valida esses hashes antes de iniciar (Stage 1/2)?
- [ ] **Cross-PRD:** O `execute-all-tasks` valida o spec-hash de dependências externas?

### 2. Carregamento sob Demanda (Skill Discovery)
- [ ] **Descoberta:** O `create-tasks` faz descoberta agnóstica via `ls .agents/skills/` ou usa lista hardcoded?
- [ ] **Conformidade de Formato:** Os regex de `## Skills Necessárias` são aplicados rigorosamente (`^- \`([a-z0-9-]+)\` — .+$`)?
- [ ] **Sincronia:** Há validação de drift entre a coluna `Skills` do `tasks.md` e os arquivos `task-*.md`?
- [ ] **Runtime:** O `execute-task` carrega apenas o que foi declarado + skill de linguagem detectada via diff?
- [ ] **Isolamento de Ciclo de Vida (execute-all-tasks):** O orquestrador carrega a skill de cada task individualmente (após os critérios de aceite) e garante a limpeza total do contexto antes de iniciar a próxima tarefa em todas as ferramentas?
- [ ] **Finalização Determinística:** O `execute-task` garante que a tarefa terminou com **zero falso positivo** (validação real vs. apenas log de conclusão)?
- [ ] **Posicionamento de Skills:** A marcação das skills ocorre obrigatoriamente logo após os critérios de aceite em todos os artefatos de tarefa?

### 3. Paridade e Portabilidade Universal
- [ ] **Zero Hardcoding:** Existem caminhos absolutos ou relativos "hardcoded" que assumem a estrutura deste repo específico?
- [ ] **Consistência de Variáveis:** O uso de `AI_TASKS_ROOT` e `AI_PRD_PREFIX` é consistente em todas as 5 skills para garantir que funcionem em qualquer subdiretório de outro projeto?
- [ ] **Bootstrap:** Se eu copiar a pasta `.agents/` para um projeto novo e vazio, as skills conseguem inicializar o fluxo (`create-prd`) sem erros de dependência de arquivos inexistentes?
- [ ] **Hooks:** Os ganchos (`pre-execute-all-tasks.sh`, `post-execute-task.sh`) são resolvidos em múltiplos caminhos (.claude, .gemini, .agents) permitindo customização local sem alterar a skill core?
- [ ] **Agents:** Existem definições específicas para cada ferramenta (`.claude/agents/`, `.gemini/agents/`, etc.) que seguem o mesmo contrato de entrada/saída?

### 4. Robustez e Segurança
- [ ] **Atomicidade:** O `execute-task` usa lock (`flock` ou rename atômico) ao editar `tasks.md`?
- [ ] **Checkpoints:** Há persistência de checkpoints em caso de crash do subagente?
- [ ] **Review Integration:** O fluxo de `review` -> `bugfix` -> `review` é respeitado com o depth limit?
- [ ] **Normalização de Formato (Antifragilidade):** O sistema tolera variações de casing (`não` vs `Não`), espaços extras ou quebras de linha em regex de validação (B01, B02)?
- [ ] **Independência de Ambiente:** O cálculo de hashes e comandos core evita dependência direta de binários locais como `sha256sum` (B03)?
- [ ] **Gestão de Recursos:** O orquestrador garante o encerramento de subagentes em caso de timeout ou falha (B04)?

## Output Esperado (Relatório de Auditoria)

1. **Score Geral (0-100):** Uma nota baseada na média dos pilares: Traceabilidade, Conformidade Procedural, Paridade e Robustez.
2. **Índice de Portabilidade:** Avaliação específica de quão "plug-and-play" o sistema é para outros repositórios.
3. **Matriz de Conformidade:** Tabela comparando suporte a features entre Claude, Gemini, Codex e Copilot.
4. **Lista de Melhorias e Bugs (Mandatório):** Itens acionáveis ordenados por prioridade (Crítico, Alta, Média, Baixa) com diagnóstico da causa raiz.
5. **Veredito de Prontidão para Produção (SEM FALSO POSITIVO):** Avaliação final se o sistema pode rodar de forma 100% autônoma ou se há "armadilhas" de formato/lógica.
6. **Veredito de Comportamento Idêntico:** Confirmação se as skills se comportam EXATAMENTE da mesma forma independentemente do projeto alvo.

## Auditoria de Prontidão (Maio 2026)

### Pontos de Melhoria e Bugs Detectados

| ID | Descrição | Criticidade | Causa Raiz |
|----|-----------|-------------|------------|
| B01 | **Drift de Skills Falso Positivo** | Crítica | A validação em `execute-task` (Etapa 2.5) é puramente textual/regex. Espaços ou quebras de linha diferentes entre `tasks.md` e o task file barram a execução, mesmo que a intenção seja a mesma. |
| B02 | **Fragilidade de Case-Sensitivity** | Alta | Flags como `Não` e `Com` em `tasks.md` exigem casing exato e acentuação. Agentes de IA tendem a alucinar `não` ou `Sim`, quebrando o loop topológico do orquestrador. |
| B03 | **Dependência de sha256sum** | Média | O cálculo de `spec-hash` depende de binários linux. Falha em ambientes Windows nativos ou containers minimalistas sem coreutils. |
| B04 | **Zombie Subagents (Timeout)** | Média | O timeout em `execute-all-tasks` é "soft". O orquestrador descarta o resultado, mas o processo do subagente continua vivo consumindo tokens/quota até o limite da sessão. |
| B05 | **Governança Hardcoded** | Baixa | A lista de skills a ignorar no auto-load (`create-tasks` Etapa 4.1) é estática. Novas skills de infra/governança precisam de atualização manual no código da skill. |

### Veredito de Produção

**ESTADO: PRONTO COM RESSALVAS**

O ecossistema é arquiteturalmente sólido e a rastreabilidade via hashes é exemplar. No entanto, **não está pronto para produção 100% autônoma "unsupervised"** devido ao excesso de "gates de formato" (Regex/Case-sensitive). 

**Para alcançar "Produção Sem Falso Positivo":**
1. Implementar normalização (case-insensitive, trim) antes de rodar os regex de validação.
2. Abstrair o cálculo de hash para um comando agnóstico (`ai-spec hash` em vez de `sha256sum`).
3. Implementar um mecanismo de sinalização de cancelamento para subagentes em caso de timeout.

---
## Instrução Adicional
Mantenha a essência do projeto: **Governança acima de automação mágica.** Não proponha soluções que escondam o estado do sistema do desenvolvedor.
