# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Sunset planejado do legacy mode (`copilotInvoker`/`codexInvoker` + `internal/wrapper`)
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RIN-05); techspec; ADR-012 (Copilot ACP nativo); ADR-013 (Codex ACP nativo); `internal/taskloop/agent.go` (`copilotInvoker`/`codexInvoker`); `internal/wrapper/wrapper.go`

## Contexto

Os caminhos legados `copilotInvoker`/`codexInvoker` (`internal/taskloop/agent.go`) e `internal/wrapper/wrapper.go` coexistem com os runtimes ACP nativos (ADR-012/ADR-013) emitindo avisos de depreciação. Manter dois caminhos por CLI **dobra a superfície de divergência** — exatamente o oposto do objetivo de paridade absoluta deste PRD. Remover, porém, é refactor amplo e arriscado se misturado à entrega de paridade.

Decisão de escopo (clarificação com o usuário): **planejar o sunset nesta techspec, não remover neste ciclo.** A remoção vira tarefa futura com release alvo e guia de migração definidos.

## Decisão

1. **Não remover** legacy mode no escopo deste PRD. Esta ADR registra critério de remoção e plano de migração.
2. **Marcação de depreciação reforçada:** garantir que todo entrypoint legado emita aviso claro e único apontando para o runtime ACP equivalente (`--runtime acp`), com link para o guia de migração.
3. **Critério de remoção (gate para a tarefa futura):**
   - Paridade RP-01..RP-04 verde nas 4 CLIs via runtime ACP (suíte `internal/parity`).
   - Guard de governança (ADR-022) ativo e estável.
   - Guia de migração publicado em `docs/`.
   - Nenhum consumidor interno (self-dogfooding) dependente do legacy.
4. **Release alvo:** próxima minor após a conclusão e estabilização das fases de paridade deste PRD (a ser fixada na tarefa de remoção; não antecipar versão aqui para não criar compromisso sem evidência).
5. **Anti-regressão durante coexistência:** legacy permanece coberto por seus testes atuais; nenhuma nova feature de paridade é adicionada ao legacy (só ao runtime ACP).

## Alternativas Consideradas

- **Remover já neste escopo.** Vantagem: reduz superfície mais cedo. Desvantagem: mistura refactor amplo com entrega de paridade, aumenta risco de regressão num único ciclo. Rejeitada (decisão do usuário).
- **Manter indefinidamente.** Vantagem: zero risco imediato. Desvantagem: divergência permanente entre dois caminhos por CLI — contraria o objetivo do PRD. Rejeitada.

## Consequências

### Benefícios Esperados

- Caminho de remoção explícito e auditável, sem comprometer a entrega de paridade.
- Sinaliza intenção aos consumidores via depreciação reforçada.

### Trade-offs e Custos

- Coexistência continua custando manutenção dupla até a remoção.
- Risco de "depreciação eterna" se a tarefa futura não for priorizada — mitigado pelo critério de remoção objetivo.

### Riscos e Mitigações

- **Risco:** consumidor externo depender do legacy. **Mitigação:** guia de migração + janela de depreciação antes da remoção.
- **Rollback:** não aplicável (nada é removido neste ciclo).

## Plano de Implementação

1. Auditar e reforçar mensagens de depreciação nos entrypoints legados.
2. Esboçar guia de migração legacy → `--runtime acp` em `docs/`.
3. Registrar a tarefa de remoção com o critério acima como pré-condição.

## Monitoramento e Validação

- Sinal de prontidão para remoção: todos os itens do critério satisfeitos.
- Critério de revisão: revisitar a cada release até a remoção efetiva.

## Impacto em Documentação e Operação

- `docs/` ganha guia de migração; AGENTS.md/`CLAUDE.md` apontam runtime ACP como caminho único recomendado.

## Revisão Futura

- Substituir esta ADR por uma ADR de remoção (Aceita) quando o critério for satisfeito.
