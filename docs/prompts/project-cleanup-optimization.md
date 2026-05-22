# Prompt Enriquecido: Limpeza e Otimização Profunda do Projeto (Criterioso)

## Prompt Original
> "analisar criteriosamente TODO o projeto e REMOVER todo 'lixo', código morto, documentação desatualizada, prompts desatualizados do projeto."

---

## Prompt Enriquecido

### Persona e Objetivo
**Atue como um Engenheiro de Staff e Arquiteto de Sistemas Especialista em Governança de IA e Eficiência de Codebase.**

Seu objetivo é realizar uma auditoria "impiedosa" e cirúrgica no repositório `ai-spec-harness` (Orchestrator). Você deve identificar e remover elementos que degradam a qualidade, aumentam o ruído de contexto ou representam débito técnico acumulado. O projeto deve ser deixado em um estado "lean", onde cada arquivo e linha de código possui uma razão clara de existir e está alinhado com a **Fonte de Verdade (`AGENTS.md`)** e os **Protocolos de 2026 (ACP/MCP)**.

### 1. Critérios de Identificação de "Lixo" e Débito

#### A. Código Morto e Legado (Prioridade Máxima)
- **Modo Wrapper Legado:** Identificar e propor a remoção de lógica em `internal/wrapper/` e comandos em `.gemini/commands/` que foram marcados como deprecados (ref: ADR-015).
- **Go Dead Code:** Funções, structs ou constantes em `internal/` que não são referenciadas em nenhum ponto de entrada ou teste.
- **Scripts Órfãos:** Scripts em `scripts/` ou `.agents/hooks/` que não são invocados por workflows, hooks ou documentação.
- **Skills Redundantes:** Verificar se há divergência ou duplicidade estática entre `.agents/skills/` e os mirrors em `.claude/`, `.gemini/`, etc., que não foram atualizados pelo `ai-spec upgrade`.

#### B. Documentação e Relatórios Obsoletos
- **Relatórios Temporários:** Remover arquivos de relatório de sessões passadas em `docs/` (ex: `report-cross-cli-validation-*.md`) que não servem mais como base de decisão.
- **ADRs em Rascunho:** Identificar ADRs em `tasks/adr/` ou `docs/adr/` que foram superadas por ADRs posteriores e não foram marcadas como `Superceded` ou removidas.
- **Tasks Concluídas/Abandonadas:** Analisar pastas em `tasks/` que não possuem atividade há muito tempo ou cujos PRs/Branches já foram mergeados (usar `gh pr list --state merged` para validar).

#### C. Prompts e Skills Desatualizados
- **Templates Legados:** Prompts em `docs/prompts/` que não seguem a estrutura de 4 blocos definida no `README.md`.
- **Inconsistência de Skill:** Skills que referenciam ferramentas ou fluxos que mudaram (ex: referências a `tasks/` em vez de `.spec/` se a migração já ocorreu).

### 2. Procedimento de Limpeza (Segurança em Primeiro Lugar)

1.  **Mapeamento de Dependências:** Antes de deletar, use `grep_search` ou `gh search code` para garantir que o alvo não é uma dependência "escondida" de algum hook ou subagente.
2.  **Validação contra AGENTS.md:** Qualquer arquivo de governança que contradiga o `AGENTS.md` deve ser atualizado ou removido.
3.  **Remoção em Lotes:** Agrupar remoções por categoria (Ex: `docs`, `internal`, `skills`) para facilitar a revisão do diff.
4.  **Preservação Mandatória:** NÃO remover `.git`, `.github/workflows`, `go.mod`, `go.sum`, `AGENTS.md`, `README.md` ou arquivos de licença.

### 3. Validação Pós-Limpeza (Definition of Done)
Após a remoção, é mandatório garantir que o sistema permanece íntegro:
- **Integridade de Go:** Rodar `make vet`, `make lint` e `make test` para garantir que nada quebrado foi deixado para trás.
- **Sanidade do Harness:** Executar `ai-spec doctor .` e `ai-spec lint .` para validar se a estrutura de governança continua válida.
- **Verificação de Spec-Drift:** Rodar `ai-spec check-spec-drift` em bundles ativos para garantir que referências a arquivos deletados não quebraram a cadeia de confiança.

### 4. Saída Esperada
Um relatório detalhado da limpeza contendo:
- **Lista de Arquivos Removidos:** Com a justificativa técnica para cada um.
- **Refatorações Realizadas:** Onde o código foi "limpo" em vez de removido.
- **Status de Validação:** Evidência de que os testes e o `doctor` passaram após a intervenção.

---

## Justificativa do Enriquecimento (Para o Usuário)

1.  **Foco em AGENTS.md:** Estabelece a fonte de verdade para evitar remoções acidentais de regras importantes.
2.  **Critérios Objetivos:** Transforma "lixo" em categorias técnicas (código morto, doc obsoleta, templates legados).
3.  **Segurança e Validação:** Adiciona gates mandatórios (`make test`, `ai-spec doctor`) para garantir que a limpeza não quebre o Orchestrator.
4.  **Uso de Ferramentas Reais:** Integra o `gh cli` para validar o status de tasks antes de removê-las, trazendo dados reais do ciclo de vida do projeto.
5.  **Alinhamento 2026:** Garante que a limpeza foque na transição para ACP/MCP e na remoção do modo wrapper legado.
