package hooks_test

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

func TestMemoryPersistHook_TypeMismatchIsObservable(t *testing.T) {
	store, _ := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	evt := hooks.SessionPostEndEvent{Summary: "wrong-type"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("Run() with mismatched Summary type must still return nil (session must not abort); got: %v", err)
	}

	if !strings.Contains(logBuf.String(), "unexpected SessionPostEndEvent.Summary type") {
		t.Errorf("log output = %q, want an observable record of the type mismatch instead of silent discard", logBuf.String())
	}
}
