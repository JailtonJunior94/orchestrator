// Package render implementa renderizadores de output para eventos ACP.
// Ver RF-11 e techspec §"Renderer (RF-11, OC #9)".
package render

import (
	"fmt"
	"io"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// HumanRenderer renderiza eventos ACP como texto legível para humanos.
// Injeta io.Writer no construtor; use io.Discard com --quiet (RF-11, decisão #18).
type HumanRenderer struct {
	out io.Writer
}

// NewHumanRenderer cria um HumanRenderer que escreve em out.
// Para --quiet, passe io.Discard; para output normal, passe os.Stdout.
func NewHumanRenderer(out io.Writer) *HumanRenderer {
	return &HumanRenderer{out: out}
}

// Render renderiza um evento para o writer. Não retorna erro (R-ERR-001):
// falhas de escrita são descartadas silenciosamente pois o writer pode ser io.Discard.
func (r *HumanRenderer) Render(evt events.Event) {
	switch evt.Kind() {
	case events.KindAgentMessage:
		if p := evt.AgentMessage(); p != nil {
			_, _ = fmt.Fprintf(r.out, "[agent] %s\n", p.Text())
		}

	case events.KindAgentThought:
		if p := evt.AgentThought(); p != nil {
			_, _ = fmt.Fprintf(r.out, "[thought] %s\n", p.Text())
		}

	case events.KindToolCallStart:
		if p := evt.ToolCallStart(); p != nil {
			_, _ = fmt.Fprintf(r.out, "[tool] %s %s\n", p.Name(), evt.ToolCallID())
		}

	case events.KindToolCallUpdate:
		if p := evt.ToolCallUpdate(); p != nil && p.Final() {
			_, _ = fmt.Fprintf(r.out, "[tool:done] %s %s\n", evt.ToolCallID(), p.Status())
		}

	// KindSessionEnd, KindRuntimeInit, KindUnknown: sem output para o usuário.
	default:
	}
}
