package hooks

import (
	"context"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/qualitygate"
)

type QualityGateHook struct {
	gate  *qualitygate.Gate
	input qualitygate.EvaluationInput
}

var _ Hook = (*QualityGateHook)(nil)

func NewQualityGateHook(gate *qualitygate.Gate, input qualitygate.EvaluationInput) *QualityGateHook {
	return &QualityGateHook{gate: gate, input: input}
}

func (h *QualityGateHook) Name() string { return "quality_gate" }

func (h *QualityGateHook) Run(ctx context.Context, evt Event) error {
	if _, ok := evt.(SessionPostEndEvent); !ok {
		return nil
	}
	if h.gate == nil {
		return nil
	}

	result := h.gate.Evaluate(ctx, h.input)
	switch result.Decision() {
	case hookcontract.DecisionBlock, hookcontract.DecisionError:
		return fmt.Errorf("quality_gate: %s (policy=%s gate=%s)", result.Reason(), result.PolicyID(), result.GateID())
	default:
		return nil
	}
}

func RegisterQualityGate(disp Dispatcher, hook Hook) {
	disp.Register(PointSessionPostEnd, hook)
}
