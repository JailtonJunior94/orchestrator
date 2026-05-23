# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** DriverID como Value Object + paridade de normalização (alias + input_mappings) nas 4 CLIs
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD `.specs/prd-paridade-cross-cli/prd.md` (RP-01, RP-03, RP-04, RIN-02); techspec `.specs/prd-paridade-cross-cli/techspec.md`; ADR-008 (29 invariantes); `internal/runtime/events/normalize.go`; `.agents/normalization-rules.yaml` (embedded em `internal/runtime/events/normalization-rules.yaml`)

## Contexto

A normalização de tool-calls é hoje driver-aware via `aliases[driverID]` e `input_mappings[driverID]`, com `driverID` circulando como `string` crua (`r.spec.ID`, `BuildNormalizedToolCall(specID, ...)`). Duas lacunas comprovadas:

- **RP-01/RIN-02:** `input_mappings` definidos só para `claude`, `codex`, `cursor`. Copilot e Gemini não têm entrada — campos de input divergem entre CLIs para a mesma task.
- **RP-04:** Gemini só aparece em `inherit_common` (herda `common_aliases`), sem tabela `aliases.gemini` explícita. Claude/Codex/Copilot têm tabela própria. A herança torna a paridade implícita e frágil a mudanças no bloco comum.

Como `driverID` é `string` solta, um valor inválido (typo, driver não suportado) degrada silenciosamente para passthrough — não há fail-fast nem fonte única de verdade dos drivers suportados. Isso conflita com R-DDD-001 (encapsular primitivo de domínio com regra) e com a doutrina de paridade *provável por teste*.

## Decisão

1. Introduzir o Value Object **`DriverID`** em `internal/runtime/specs` (pacote que já é dono do conceito de driver):
   - Construtor `ParseDriverID(string) (DriverID, error)` que valida contra o conjunto canônico (`claude`, `codex`, `copilot`, `gemini`); valor desconhecido retorna `ErrUnknownDriver`.
   - Imutável, identidade por valor, método `String()`.
   - `specs.Spec.ID` continua `string` no wire/JSON (sem quebra), mas o runtime resolve `DriverID` na fronteira (ao construir a Spec) e propaga o VO internamente.
2. **RP-04:** adicionar tabela `aliases.gemini` explícita em `normalization-rules.yaml`, materializando os aliases que hoje vêm de `common_aliases`. Manter `inherit_common` apenas como fallback documentado; a tabela explícita vence (já garantido por `resolveInherit`, que não sobrescreve entrada existente).
3. **RP-01/RIN-02:** adicionar `input_mappings.copilot` e `input_mappings.gemini`. Quando um driver é comprovadamente no-op, registrar entrada vazia com comentário `# no-op verificado` em vez de ausência (torna o no-op explícito e testável).
4. **RP-03:** promover as invariantes ADR-008 a uma suíte de paridade executável (`internal/parity`) que asserta, para uma task fixture, igualdade do conjunto de `normalized_name` e da forma de evento entre os 4 drivers.

`RawName`/`RawInput` permanecem byte-identical (garantia já existente em `NormalizedToolCall`); esta decisão não toca o caminho `--no-normalize`.

## Alternativas Consideradas

- **Manter `driverID string` e só completar o YAML.** Vantagem: menor diff. Desvantagem: perpetua passthrough silencioso para driver inválido e não cria fonte única dos drivers suportados; viola R-DDD-001. Rejeitada por não fechar o gap de fail-fast.
- **Eliminar `inherit_common` e exigir tabela explícita para todos.** Vantagem: zero ambiguidade. Desvantagem: remove o mecanismo de herança que espelha o Compozy e quebraria projetos com override que dependem do comum. Rejeitada por blast radius desnecessário — herança como fallback é suficiente.
- **Enum gerado por codegen.** Overhead de tooling desproporcional para 4 valores estáveis. Rejeitada.

## Consequências

### Benefícios Esperados

- Fail-fast em driver desconhecido na fronteira, com erro tipado (`ErrUnknownDriver`).
- Paridade de normalização *provável por teste* (RP-03), não só documentada — capitaliza ADR-008.
- Fonte única dos drivers suportados (`specs`), reduzindo divergência ao adicionar nova CLI.

### Trade-offs e Custos

- Refactor de assinatura interna (`string` → `DriverID`) em pontos do `runtime`. Mitigado mantendo `Spec.ID` string e convertendo só na fronteira.
- Tabelas YAML maiores (gemini/copilot explícitos). Custo marginal de manutenção.

### Riscos e Mitigações

- **Risco:** override de projeto (`.agents/normalization-rules.yaml`) sem as novas chaves vira regressão. **Mitigação:** loader já é tolerante a campos ausentes (passthrough); suíte RP-03 roda contra o embedded default.
- **Rollback:** `--no-normalize` recupera comportamento pré-normalização byte-identical.

## Plano de Implementação

1. Criar `DriverID` VO + `ParseDriverID` + `ErrUnknownDriver` em `specs`.
2. Resolver `DriverID` ao construir cada Spec; propagar no `ACPRunner`/`runEventLoop`.
3. Completar `normalization-rules.yaml`: `aliases.gemini`, `input_mappings.copilot`, `input_mappings.gemini`.
4. Suíte de paridade RP-03 em `internal/parity` derivada de ADR-008.
5. Golden tests por driver para `input_mappings`.

## Monitoramento e Validação

- Gate: `make test` + suíte `internal/parity` verde para os 4 drivers.
- Sinal de sucesso: mesma task fixture → mesmo conjunto de `normalized_name` e forma de evento nas 4 CLIs.
- Critério de revisão: adição de 5ª CLI ou mudança no schema de `normalization-rules.yaml` (`version` ≠ 1).

## Impacto em Documentação e Operação

- Atualizar `CLAUDE.md`/`AGENTS.md` (seção Normalização) com a tabela Gemini explícita.
- Atualizar `docs/` de normalização se existente.

## Revisão Futura

- Revisitar ao adicionar nova CLI ou ao bump de `version` do schema de regras.
