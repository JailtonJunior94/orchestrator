# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Adoção do Agent Client Protocol (ACP) via `coder/acp-go-sdk` para invocação de agentes
- **Data:** 2026-05-20
- **Status:** Aceita
- **Decisores:** Mantenedores do `ai-spec-harness`
- **Relacionados:**
  - PRD: `.specs/prd-acp-runtime-claude/prd.md`
  - ADRs anteriores: [001](001-go-embed-baseline.md), [002](002-fake-filesystem-testes.md), [005](005-skills-lock-sha256.md)
  - Referência externa: catálogo `internal/core/agent/registry_specs.go` do projeto compozy/compozy

## Contexto

O `ai-spec-harness` invoca agentes de IA (Claude, Codex, Gemini, Copilot) por meio de chamadas `exec.Cmd` síncronas em `internal/taskloop/agent.go`. O fluxo é: monta prompt a partir de template embutido, chama o binário do agente como subprocesso, espera ele encerrar, captura `stdout`/`stderr`/exit code, classifica resultado e grava evidência.

Esse modelo impõe três limitações concretas:

1. **Cegueira durante a execução.** Não sabemos se o agente está raciocinando, executando uma tool ou travado. Só vemos o final.
2. **Cancelamento grosseiro.** Só o timeout absoluto da task pode interromper. Um agente vivo mas mudo (loop infinito de raciocínio sem progresso) consome o timeout inteiro antes de ser cancelado.
3. **Inviabilidade de features futuras.** TUI ao vivo, execução concorrente com observabilidade compartilhada, retomada de sessão (attach/watch) e batching com retry granular dependem de granularidade que o `exec.Cmd` one-shot não dá.

Projetos comparáveis no ecossistema — em especial o `compozy/compozy` — resolvem isso adotando o **Agent Client Protocol (ACP)**, um protocolo JSON-RPC bidirecional sobre stdio entre o host (o orquestrador) e o agente. O compozy usa `github.com/coder/acp-go-sdk` e converte os `SessionUpdate` recebidos para um modelo interno (`internal/core/agent/acp_convert.go`), o que habilita streaming, watchdog de atividade, TUI bubbletea e attach/watch via daemon.

Esta decisão estabelece a adoção do mesmo protocolo, partindo de **um** runtime (Claude) e atrás de **uma** flag opt-in (`--runtime=acp`), como base técnica para evoluções subsequentes.

## Decisão

Adotar o **Agent Client Protocol (ACP)** como protocolo de comunicação suportado entre o `ai-spec-harness` e agentes de IA executados como subprocessos, usando o SDK Go oficial **`github.com/coder/acp-go-sdk`**.

Escopo desta decisão:

- Introduzir um pacote novo `internal/runtime/` que abstrai sessões ACP, eventos traduzidos (`runtime.Event`), registry de specs por runtime (`Spec`+`Launcher`) e activity watchdog.
- Implementar o cliente ACP apenas para o runtime **Claude** nesta primeira iteração (com binário canônico `claude-agent-acp` e fallback `npx --yes @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA>`, onde `<VERSAO_PINADA>` é uma constante Go atualizada via processo `audit/`).
- Expor o comportamento via flag `--runtime=acp` em `ai-spec task-loop`; default permanece `legacy` (caminho atual baseado em `exec.Cmd`).
- O caminho legacy continua suportado e testado durante a janela de coexistência. A remoção do legacy depende de decisão futura, registrada em ADR separado, quando todos os runtimes hoje suportados tiverem cobertura ACP.

Esta decisão **não** abrange: outros runtimes (Codex, Gemini etc.), TUI, daemon, retries com backoff, formato hierárquico de config, memória entre runs. Cada um exigirá ADR/PRD próprio.

## Alternativas Consideradas

### Alternativa 1: Implementar nosso próprio protocolo JSON-RPC sobre stdio

- **Descrição:** Definir um schema de mensagens próprio entre o `ai-spec-harness` e um wrapper que escreveríamos para cada agente.
- **Vantagens:** Sem dependência externa; controle total sobre semântica.
- **Desvantagens:** Reescreve trabalho que o ACP já fez; cada novo runtime precisaria de um wrapper customizado; o ecossistema (Anthropic, Zed, Cursor, OpenCode etc.) está convergindo para ACP, então iríamos contra a corrente.
- **Motivo de rejeição:** Custo de manutenção desproporcional ao benefício; perderíamos a possibilidade de aproveitar binários já existentes (`claude-agent-acp`, `codex-acp` etc.).

### Alternativa 2: Manter `exec.Cmd` e adicionar streaming via parsing de stdout

- **Descrição:** Forçar o agente a emitir JSON estruturado no stdout (uma linha por evento) e fazer parsing no harness, mantendo a interface `AgentInvoker` atual.
- **Vantagens:** Mudança mínima; sem dependência nova.
- **Desvantagens:** Frágil (qualquer linha não-JSON quebra o parser); não é padrão de mercado; não dá canal bidirecional para `requestPermission`/cancel granular; agentes diferentes geram formatos diferentes — voltaríamos a escrever um adapter por agente.
- **Motivo de rejeição:** Resolve sintoma, não causa raiz; não habilita features futuras (attach, retomada, permissões).

### Alternativa 3: Adotar ACP, mas implementar o cliente do zero sem `coder/acp-go-sdk`

- **Descrição:** Falar o protocolo ACP diretamente, escrevendo encoder/decoder JSON-RPC próprios.
- **Vantagens:** Sem dependência externa; controle de versionamento.
- **Desvantagens:** Custo alto de implementação inicial e de manutenção (rastrear evolução do protocolo); risco de divergência sutil que só aparece em produção; o SDK oficial já passa por testes do ecossistema (compozy é um stress test real).
- **Motivo de rejeição:** Não há ganho técnico mensurável; o custo de manter um cliente próprio supera o custo de carregar a dependência.

### Alternativa 4: Postergar a decisão e investir em outros eixos antes

- **Descrição:** Manter o status quo e priorizar Spec-Hash, telemetria, novos skills.
- **Vantagens:** Zero risco; foco em consolidação do que já existe.
- **Desvantagens:** Trava a evolução do `task-loop` (TUI, batch, attach) por tempo indeterminado; aumenta a divergência em relação a ferramentas comparáveis.
- **Motivo de rejeição:** A decisão de adotar ACP é cara mas reversível por flag; adiá-la não reduz custo, só atrasa benefícios.

## Consequências

### Benefícios Esperados

- **Observabilidade evento a evento:** mensagens, raciocínio e tool calls do agente ficam visíveis em tempo real e auditáveis via `events.jsonl`.
- **Cancelamento responsivo:** activity watchdog detecta travamentos vivos (sem output) em até `--activity-timeout` (default 120s), em vez de esperar o timeout absoluto.
- **Base para features de alto valor:** TUI, execução concorrente com observabilidade compartilhada e attach/watch passam a ser viáveis incrementalmente.
- **Alinhamento com ecossistema:** o protocolo ACP é o caminho que Anthropic, Zed, Cursor e OpenCode já adotaram. Reduz custo futuro de integrar novos runtimes (cada um já fala ACP).
- **Evidência mais rica para governança:** `events.jsonl` e `tool_calls.md` por task fortalecem a invariante de "evidência obrigatória" do AGENTS.md.

### Trade-offs e Custos

- **Dependência Go nova de longo prazo:** `github.com/coder/acp-go-sdk` passa a ser dependência direta no `go.mod`. Upgrades exigem auditoria documentada (`audit/`).
- **Complexidade adicional no código:** três módulos novos (`internal/runtime/`, `internal/runtime/acpfake/`, conversão de eventos) e um caminho dual de invocação (legacy + ACP) coexistindo.
- **Custo de manutenção dual:** enquanto legacy não for removido, mudanças no `task-loop` precisam ser validadas nos dois caminhos.
- **Tempo de cold-start adicional:** o fallback `npx --yes @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA>` adiciona segundos no primeiro uso quando o binário canônico não está instalado. Mitigado por cache do path resolvido e versão pinada (sem resolução `@latest`).

### Riscos e Mitigações

- **Risco:** `coder/acp-go-sdk` introduz breaking changes entre versões.
  - **Impacto:** Build quebra; comportamento do harness regride.
  - **Mitigação:** Pin de versão exato no `go.mod`; toda subida de versão exige decisão registrada em `audit/` (template `.specs/templates/skill-upgrade-decision.md`); camada de tradução `internal/runtime/convert.go` isolada para absorver mudanças.
  - **Rollback:** Reverter para a versão pinada anterior; default `--runtime=legacy` continua funcional independente do estado do SDK.

- **Risco:** O `claude-agent-acp` evolui e passa a exigir bootstrap args específicos (modelo, access mode etc.) que hoje não enviamos.
  - **Impacto:** Sessão falha em iniciar ou comporta-se de forma inesperada.
  - **Mitigação:** Spec do runtime (`internal/runtime/specs/claude.go`) tem ponto de extensão `BootstrapArgs(...)` espelhando o registry do compozy, permitindo evoluir sem mexer no cliente.
  - **Rollback:** Cair para o caminho legacy via flag.

- **Risco:** Fake ACP server diverge do servidor real e mascara bugs.
  - **Impacto:** Testes verdes mas integração real quebra.
  - **Mitigação:** Manter `tests/runtime/live/` rodando opt-in com `AI_SPEC_ACP_LIVE=1`; rodar em CI nightly se possível; gerar golden tests do shape dos eventos para detectar drift do schema.
  - **Rollback:** Não aplicável (teste, não produção); ajustar fake conforme drift detectado.

- **Risco:** Default `--runtime=legacy` permanecer indefinidamente, gerando dívida técnica de manter os dois caminhos.
  - **Impacto:** Custo de manutenção dual cresce; novos features só são adicionados em um lado.
  - **Mitigação:** Definir critério explícito de virada do default em ADR futuro (ex.: "quando 4 dos 4 tools tiverem cobertura ACP e 30 dias sem regressão"); deprecar o legacy por release.
  - **Rollback:** Reverter a flag default.

## Plano de Implementação

1. **Aprovar PRD `.specs/prd-acp-runtime-claude/prd.md` e este ADR** (status `Aceita`).
2. **Gerar techspec** em `.specs/prd-acp-runtime-claude/techspec.md` via skill `create-technical-specification`, detalhando:
   - Estrutura dos pacotes `internal/runtime/`, `internal/runtime/specs/`, `internal/runtime/acpfake/`.
   - Tipo `runtime.Event` e tabela de conversão de `acp.SessionUpdate`.
   - Contrato do fake server (subset de eventos que ele consegue emitir).
3. **Decompor em tasks** via skill `create-tasks`, com `dependencies` no frontmatter para refletir DAG: spec/registry → cliente ACP → watchdog → conversão de eventos → integração no `task-loop` → fake server → testes.
4. **Implementar tasks incrementalmente** com `execute-task`, gerando `execution_report.md` por tarefa.
5. **Pinar `github.com/coder/acp-go-sdk`** no `go.mod` com versão **stable tagged** exata (sem pseudo-version de commit, sem Renovate/Dependabot até o SDK atingir 1.0); pinar a versão npm equivalente em `internal/runtime/specs/claude.go`. Primeiro upgrade de qualquer um dos dois exige `audit/` decision file.
6. **Executar suite completa** com e sem `AI_SPEC_ACP_LIVE=1` antes do merge.
7. **Critério de adoção concluída:** RF-01 a RF-15 do PRD atendidos, `ai-spec task-loop --tool claude --runtime acp` rodando em ambiente real, e `make verify` passando nos dois caminhos.

## Monitoramento e Validação

- **Métricas (telemetria opt-in `GOVERNANCE_TELEMETRY=1`):** `runtime` e `events_count` por execução de `task-loop`. Após 30 dias, comparar taxa de `cancel_reason=activity_timeout` entre `legacy` e `acp` para validar que o watchdog não cancela em excesso.
- **Sinais de regressão a observar:**
  - Aumento de execuções que terminam em `cancel_reason=activity_timeout` indica `--activity-timeout` default mal calibrado.
  - Crescimento da contagem de `unknown` events no `events.jsonl` indica drift do SDK ou do servidor; gatilho para revisão.
  - Falhas no fake server vs. live test apontam divergência de fidelidade.
- **Critérios de sucesso:** PRD atendido, ambos os caminhos estáveis por uma release, sem necessidade de hotfix relacionado.
- **Critério para revisão:** o tema "remover legacy / promover ACP a default" deve ser tratado em ADR separado quando os outros runtimes (Codex, Gemini, Copilot) tiverem cobertura ACP.

## Impacto em Documentação e Operação

- **`README.md`:** adicionar seção curta explicando `--runtime` e como instalar `claude-agent-acp`.
- **`AGENTS.md`:** referenciar este ADR na lista de ADRs do projeto.
- **`CLAUDE.md`, `CODEX.md`, `COPILOT.md`, `GEMINI.md`:** sem alteração obrigatória nesta fase (mudança é interna).
- **`docs/telemetry-feedback-cycle.md`:** documentar os novos campos `runtime` e `events_count`.
- **Onboarding / runbook:** atualizar com troubleshooting de "binário não encontrado" e como configurar o fallback `npx`.
- **CI:** considerar adicionar job opcional com `AI_SPEC_ACP_LIVE=1` em nightly.

## Revisão Futura

- **Marco de revisão:** quando o segundo runtime (provavelmente Codex) for adicionado via ACP. Esta decisão deve ser reavaliada à luz da experiência operacional com Claude.
- **Eventos que invalidam premissas:**
  - `coder/acp-go-sdk` ser arquivado ou perder mantenedor ativo.
  - Surgimento de um protocolo concorrente amplamente adotado (improvável dado o momento, mas possível).
  - Mudanças contratuais do `claude-agent-acp` que exijam reescrever a camada de conversão por completo.
- **Condição para substituição por novo ADR:** decisão de remover o caminho legacy (vira "ADR-010: deprecação do runtime legacy"); ou decisão de trocar o SDK ACP por implementação própria.
