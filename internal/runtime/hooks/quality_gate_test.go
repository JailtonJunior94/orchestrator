package hooks

import (
	"context"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/qualitygate"
)

type recordingDispatcher struct {
	registrations map[string][]Hook
}

func newRecordingDispatcher() *recordingDispatcher {
	return &recordingDispatcher{registrations: make(map[string][]Hook)}
}

func (d *recordingDispatcher) Register(point string, hook Hook) {
	d.registrations[point] = append(d.registrations[point], hook)
}

func (d *recordingDispatcher) Dispatch(ctx context.Context, point string, evt Event) error {
	for _, hook := range d.registrations[point] {
		if err := hook.Run(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

func TestRegisterQualityGate_RegistersOnlyOnSessionPostEnd(t *testing.T) {
	disp := newRecordingDispatcher()
	hook := NewQualityGateHook(nil, qualitygate.EvaluationInput{})

	RegisterQualityGate(disp, hook)

	if len(disp.registrations[PointSessionPostEnd]) != 1 {
		t.Fatalf("expected quality gate registered exactly once on %s, got %d", PointSessionPostEnd, len(disp.registrations[PointSessionPostEnd]))
	}
	if len(disp.registrations[PointToolCallPreDispatch]) != 0 {
		t.Fatalf("quality gate must never register on %s (RF-25)", PointToolCallPreDispatch)
	}
	if len(disp.registrations[PointToolCallPostComplete]) != 0 {
		t.Fatalf("quality gate must never register on %s (RF-25)", PointToolCallPostComplete)
	}
	for _, point := range AllPoints() {
		if point == PointSessionPostEnd {
			continue
		}
		if len(disp.registrations[point]) != 0 {
			t.Fatalf("quality gate must never register on %s (RF-25 - event adequado apenas)", point)
		}
	}
}

func TestQualityGateHook_IgnoresEventsOtherThanSessionPostEnd(t *testing.T) {
	hook := NewQualityGateHook(nil, qualitygate.EvaluationInput{})

	if err := hook.Run(context.Background(), ToolCallEvent{Phase: "pre_dispatch"}); err != nil {
		t.Fatalf("unexpected error for wrong event: %v", err)
	}
}

type emptyToolchainProvider struct{}

func (emptyToolchainProvider) Detect(string) detect.ToolchainResult {
	return detect.ToolchainResult{}
}

func TestQualityGateHook_BlockDecisionAbortsWithReason(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	gate := qualitygate.NewGate(
		qualitygate.NewDefaultPolicyLoader(fakeFS),
		emptyToolchainProvider{},
		qualitygate.NewShellExecutor(),
		nil,
		nil,
	)
	hook := NewQualityGateHook(gate, qualitygate.EvaluationInput{
		ProjectDir: "/project",
		TaskID:     "10.0",
		TaskType:   qualitygate.DefaultTaskType,
		Risk:       qualitygate.DefaultRisk,
	})

	err := hook.Run(context.Background(), SessionPostEndEvent{})
	if err == nil {
		t.Fatalf("expected error when quality gate cannot resolve a stack (ERROR decision)")
	}
}
