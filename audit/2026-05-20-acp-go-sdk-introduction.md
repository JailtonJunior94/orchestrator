# Decisão de Upgrade — coder/acp-go-sdk

## Metadados

- **Skill:** `coder/acp-go-sdk`
- **Versão anterior:** -
- **Versão nova:** v0.13.0
- **Data:** 2026-05-20
- **Responsável:** JailtonJunior94

## Motivador

Introdução da dependência `github.com/coder/acp-go-sdk` para habilitar o runtime ACP no
`ai-spec-harness`, conforme PRD `tasks/prd-acp-runtime-claude/prd.md` (Restrição Técnica de Alto
Nível: "Dependência nova obrigatória") e ADR-009 (`tasks/adr/009-acp-protocol-adoption.md`).

A versão v0.13.0 é a última versão stable com tag semântica disponível em 2026-05-20,
verificada via `go list -m -versions github.com/coder/acp-go-sdk`. Não há pseudo-version
de commit nem `replace` no `go.mod`. Upgrades subsequentes exigem nova entrada em `audit/`.

Renovate/Dependabot não deve ser habilitado para esta dependência enquanto o SDK não atingir
v1.0 estável (ADR-009 §"Plano de Implementação").

## Critério de Aceitação

- `make verify` passa com a dependência pinada.
- RF-04 atendido nas tasks subsequentes (4.0 e 8.0): cliente ACP usa o SDK para abrir sessão,
  enviar prompt e consumir stream de `SessionUpdate`.
- `go list -m github.com/coder/acp-go-sdk` retorna `github.com/coder/acp-go-sdk v0.13.0`.
- Constante `ClaudeSDKVersion` em `internal/runtime/specs/claude.go` vale `"v0.13.0"`,
  sincronizada por `scripts/sync-acp-sdk-version.sh`.

## Riscos

- O SDK está em fase pré-1.0 (maior probabilidade de breaking changes entre tags).
  Mitigação: pin estrito + camada `internal/runtime/events/convert.go` isolada como único
  ponto de conversão ACP→Event.
- `go mod tidy` remove dependências não usadas. Mitigação: `internal/runtime/client/client.go`
  tem import `_ "github.com/coder/acp-go-sdk"` até a implementação real (task 8.0) ser entregue.

## Resultado

- [x] `go.mod` contém `require github.com/coder/acp-go-sdk v0.13.0`
- [x] `go.sum` consistente após `go mod tidy`
- [x] `ClaudeSDKVersion` sincronizada para `"v0.13.0"` via `scripts/sync-acp-sdk-version.sh`
- [x] `make verify` passa
- [x] Registro salvo em `audit/2026-05-20-acp-go-sdk-introduction.md`
