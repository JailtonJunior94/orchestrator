package specs

import "testing"

func TestContextWindow_Class(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		maxTokens int
		want      WindowClass
	}{
		// F1: zero-value deve retornar WindowStandard (invariante crítica).
		{"zero_value_F1", 0, WindowStandard},
		// Abaixo do limiar.
		{"below_threshold", 999_999, WindowStandard},
		// Exatamente no limiar.
		{"at_threshold_1M", 1_000_000, WindowLarge},
		// Acima do limiar.
		{"above_threshold", 2_000_000, WindowLarge},
		// Pequeno valor positivo.
		{"small_positive", 1, WindowStandard},
		// Valor típico de janela padrão (200K tokens).
		{"standard_200k", 200_000, WindowStandard},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := ContextWindow{MaxTokens: tc.maxTokens}
			got := w.Class()
			if got != tc.want {
				t.Errorf("ContextWindow{MaxTokens:%d}.Class() = %v; want %v", tc.maxTokens, got, tc.want)
			}
		})
	}
}

func TestContextWindow_ZeroValue_IsWindowStandard(t *testing.T) {
	t.Parallel()

	// Garantia explícita de zero-regressão F1.
	var w ContextWindow
	if w.Class() != WindowStandard {
		t.Errorf("ContextWindow{}.Class() = %v; want WindowStandard (F1 invariant)", w.Class())
	}
}

func TestWindowClass_ClosedSet(t *testing.T) {
	t.Parallel()

	// WindowStandard deve ser o zero-value do enum.
	if WindowStandard != 0 {
		t.Errorf("WindowStandard = %d; want 0 (zero-value)", int(WindowStandard))
	}

	// WindowLarge deve ser distinto de WindowStandard.
	if WindowLarge == WindowStandard {
		t.Error("WindowLarge e WindowStandard não devem ser iguais")
	}
}
