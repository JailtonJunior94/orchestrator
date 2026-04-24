package taskloop

// keyAction identifica uma acao de teclado disponivel na TUI do task-loop.
type keyAction int

const (
	keyQuit     keyAction = iota // q ou ctrl+c: encerra a TUI
	keyPause                     // p: alterna pausa visual (sem sinal ao Service nesta fase — ADR-005)
	keySkip                      // s: skip visual (sem efeito real nesta fase — ADR-005)
	keyTabFocus                  // tab: cicla foco entre os tres paineis
	keyNone                      // tecla sem acao mapeada
)

// resolveKeyAction mapeia a string de tecla para a keyAction correspondente.
// Retorna keyNone para teclas nao mapeadas.
func resolveKeyAction(key string) keyAction {
	switch key {
	case "ctrl+c", "q":
		return keyQuit
	case "p":
		return keyPause
	case "s":
		return keySkip
	case "tab":
		return keyTabFocus
	default:
		return keyNone
	}
}
