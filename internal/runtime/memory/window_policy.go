// window_policy.go — domain service stateless de política de janela (ADR-023).
// Decide os limites de compactação da memória 2-tier com base na WindowClass da CLI ativa.
// WindowStandard ⇒ limites F1 (150/12KB · 200/16KB); WindowLarge ⇒ limites ampliados.
// Zero-value de WindowClass (WindowStandard) preserva comportamento F1 exato.
package memory

import "github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"

// Limites ampliados para WindowLarge (ADR-023).
// Aproximadamente 3× os defaults F1 para evitar compactação prematura em janelas de 1M+ tokens.
const (
	LargeWorkflowLineLimit = 450       // 3× DefaultWorkflowLineLimit (150)
	LargeWorkflowByteLimit = 36 * 1024 // 3× DefaultWorkflowByteLimit (12KB)
	LargeTaskLineLimit     = 600       // 3× DefaultTaskLineLimit (200)
	LargeTaskByteLimit     = 48 * 1024 // 3× DefaultTaskByteLimit (16KB)
)

// WindowPolicy é o domain service stateless que mapeia WindowClass → Limits (ADR-023).
// Implementação: função pura (stateless); satisfaz interface implícita se necessário no futuro.
type WindowPolicy struct{}

// DefaultWindowPolicy é a instância de produção do domain service (stateless, compartilhável).
var DefaultWindowPolicy = WindowPolicy{}

// LimitsFor retorna os limites de compactação ajustados para a WindowClass dada.
// base é o ponto de partida; quando zero-value, os campos são preenchidos pelos defaults F1 antes
// do ajuste para evitar limites zero acidentais.
//
//   - WindowStandard (ou zero-value) ⇒ base com defaults F1 aplicados (sem alteração F1).
//   - WindowLarge ⇒ limites ampliados (≥3× F1) para não compactar prematuramente janelas ≥1M.
func (p WindowPolicy) LimitsFor(class specs.WindowClass, base Limits) Limits {
	return p.LimitsForWithOverride(class, base, false)
}

// LimitsForWithOverride retorna os limites de compactação considerando se base veio de
// override explicito do usuario. Para WindowLarge, overrides explicitos prevalecem.
func (p WindowPolicy) LimitsForWithOverride(class specs.WindowClass, base Limits, explicit bool) Limits {
	// Aplicar defaults F1 nos campos zero antes de qualquer ajuste.
	base = NewCatalog().applyDefaultsToLimits(base)

	if class != specs.WindowLarge {
		// WindowStandard (zero-value incluso): retornar limites base com defaults F1.
		return base
	}
	if explicit {
		return base
	}

	// WindowLarge: usar limites ampliados independentemente do base fornecido,
	// garantindo que o store nunca compacte nos limiares conservadores F1.
	return Limits{
		WorkflowLines: LargeWorkflowLineLimit,
		WorkflowBytes: LargeWorkflowByteLimit,
		TaskLines:     LargeTaskLineLimit,
		TaskBytes:     LargeTaskByteLimit,
	}
}

// applyDefaultsToLimits preenche campos zero-value com os defaults F1.
// Espelha a lógica de prepareMemoryStore em runner.go (sem duplicar a fonte única em store.go).
func (c *Catalog) applyDefaultsToLimits(l Limits) Limits {
	if l.WorkflowLines == 0 {
		l.WorkflowLines = DefaultWorkflowLineLimit
	}
	if l.WorkflowBytes == 0 {
		l.WorkflowBytes = DefaultWorkflowByteLimit
	}
	if l.TaskLines == 0 {
		l.TaskLines = DefaultTaskLineLimit
	}
	if l.TaskBytes == 0 {
		l.TaskBytes = DefaultTaskByteLimit
	}
	return l
}
