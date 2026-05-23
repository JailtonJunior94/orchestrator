package specs

// WindowClass é um enum tipado que classifica a janela de contexto de um driver.
// Conjunto fechado: WindowStandard | WindowLarge.
// Zero-value é WindowStandard (preserva comportamento F1 — nenhuma regressão).
type WindowClass int

const (
	// WindowStandard representa janelas de contexto padrão (< 1M tokens).
	// Zero-value intencional: ContextWindow{}.Class() == WindowStandard (F1).
	WindowStandard WindowClass = iota

	// WindowLarge representa janelas de contexto grandes (>= 1M tokens).
	// Exemplo: Gemini com janela de 1M+ tokens.
	WindowLarge
)

// ContextWindow é um Value Object que descreve a janela de contexto de um driver ACP.
// Imutável; classe derivada de forma estável via Class().
// Zero-value (MaxTokens == 0) resulta em WindowStandard (F1 — zero-regressão).
type ContextWindow struct {
	// MaxTokens é o limite máximo de tokens da janela de contexto.
	// 0 indica janela padrão (zero-value seguro).
	MaxTokens int
}

// largeTokenThreshold define o limiar (em tokens) acima do qual a janela é classificada como WindowLarge.
// 1_000_000 tokens (1M) é o referencial de janela grande (ADR-023).
const largeTokenThreshold = 1_000_000

// Class retorna a classe da janela de contexto derivada de MaxTokens.
// Zero-value (MaxTokens == 0) retorna WindowStandard (invariante F1).
func (w ContextWindow) Class() WindowClass {
	if w.MaxTokens >= largeTokenThreshold {
		return WindowLarge
	}
	return WindowStandard
}
