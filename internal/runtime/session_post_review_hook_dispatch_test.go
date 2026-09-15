package runtime_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type failingSessionPostReviewHook struct {
	err error
}

func (h *failingSessionPostReviewHook) Name() string { return "test_session_post_review_failure" }

func (h *failingSessionPostReviewHook) Run(_ context.Context, _ hooks.Event) error {
	return h.err
}

func TestACPRunner_SessionPostReviewDispatchFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("tarefa com auto-review e falha de hook pos-review").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "## Review\n\nNo blocking issues found.\n\nverdict: APPROVED\n", nil
	}

	hookErr := errors.New("session.post_review sink unavailable")

	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactoryForReview{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(&fakePersistenceFactoryForReview{}),
		airuntime.NewCatalog().WithRenderer(&fakeRendererForReview{}),
		airuntime.NewCatalog().WithReviewOutputFn(reviewFn),
		airuntime.NewCatalog().WithSessionPostReviewTestHook(&failingSessionPostReviewHook{err: hookErr}),
	)

	job := airuntime.Job{
		Prompt:      "tarefa com auto-review",
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true,
	}

	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	summary, err := runner.Run(ctx, job)

	if err != nil {
		t.Fatalf("Run: session must not abort on session.post_review dispatch failure; got err=%v", err)
	}

	if summary.ReviewStatus != "ok" {
		t.Fatalf("summary.ReviewStatus = %q, want ok (hook dispatch failure must not affect review verdict)", summary.ReviewStatus)
	}

	if len(summary.HookDispatchErrors) == 0 {
		t.Fatal("summary.HookDispatchErrors is empty; session.post_review dispatch failure must be composed into the Summary")
	}
	if !strings.Contains(summary.HookDispatchErrors[0], hookErr.Error()) {
		t.Errorf("summary.HookDispatchErrors[0] = %q, want it to reference %q", summary.HookDispatchErrors[0], hookErr.Error())
	}

	if !strings.Contains(logBuf.String(), "session.post_review hook dispatch failed") {
		t.Errorf("log output = %q, want explicit log of the session.post_review dispatch failure", logBuf.String())
	}
}
