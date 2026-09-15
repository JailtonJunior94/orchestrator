//go:build integration

package durable_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

const (
	envHelperFlag    = "AISPEC_LOCK_HELPER"
	envHelperTasks   = "AISPEC_LOCK_TASKS_DIR"
	envHelperFactID  = "AISPEC_LOCK_FACT_ID"
	helperTestSelect = "-test.run=^TestHelperProcessConsolidate$"
)

func TestHelperProcessConsolidate(t *testing.T) {
	if os.Getenv(envHelperFlag) != "1" {
		t.Skip("helper process only, invoked via subprocess reinvocation")
	}

	tasksDir := os.Getenv(envHelperTasks)
	factID := os.Getenv(envHelperFactID)
	if tasksDir == "" || factID == "" {
		t.Fatal("multiprocess helper: missing AISPEC_LOCK_TASKS_DIR or AISPEC_LOCK_FACT_ID")
	}

	fsys := fs.NewOSFileSystem()
	layer := durable.NewLayer(fsys)
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
	catalog := durable.NewCatalog()

	content := fmt.Sprintf("durable fact written by real process %s", factID)
	key, err := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "multiprocess-helper", Subject: factID})
	if err != nil {
		t.Fatalf("multiprocess helper %s: derive semantic key: %v", factID, err)
	}
	fact := durable.Fact{
		Identity:   durable.Identity{Key: key, Hash: catalog.HashContent(content)},
		Content:    content,
		Durability: durable.DurabilityPRD,
		Origin: durable.FactOrigin{
			Session: factID,
			CLI:     "helper",
			Task:    "",
			Date:    time.Now().UTC().Format(time.RFC3339),
		},
	}

	ctx := context.Background()
	deadline := time.Now().Add(15 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, consErr := layer.Consolidate(ctx, scope, []durable.Fact{fact}); consErr == nil {
			return
		} else if errors.Is(consErr, durable.ErrLayerLocked) {
			lastErr = consErr
			time.Sleep(5 * time.Millisecond)
			continue
		} else {
			t.Fatalf("multiprocess helper %s: unexpected non-lock error: %v", factID, consErr)
		}
	}
	t.Fatalf("multiprocess helper %s: exceeded retry deadline waiting for layer lock, last error: %v", factID, lastErr)
}

func TestTwoRealProcessesCompeteForSameLayer(t *testing.T) {
	tasksDir := t.TempDir()

	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
	activePath, pathErr := scope.ActivePath()
	if pathErr != nil {
		t.Fatalf("resolve PRD scope active path: %v", pathErr)
	}
	fsys := fs.NewOSFileSystem()
	if err := fsys.MkdirAll(filepath.Dir(activePath)); err != nil {
		t.Fatalf("pre-create memory directory so both subprocesses can open the lock file: %v", err)
	}

	const processCount = 2
	results := make(chan error, processCount)
	for i := 0; i < processCount; i++ {
		factID := fmt.Sprintf("proc-%d", i)
		go func(id string) {
			cmd := exec.Command(os.Args[0], helperTestSelect) //nolint:gosec
			cmd.Env = append(os.Environ(),
				envHelperFlag+"=1",
				envHelperTasks+"="+tasksDir,
				envHelperFactID+"="+id,
			)
			output, runErr := cmd.CombinedOutput()
			if runErr != nil {
				results <- fmt.Errorf("subprocess %s failed: %w\noutput:\n%s", id, runErr, output)
				return
			}
			results <- nil
		}(factID)
	}

	for i := 0; i < processCount; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}

	layer := durable.NewLayer(fsys)

	facts, _, err := layer.Read(context.Background(), scope)
	if err != nil {
		t.Fatalf("read consolidated layer page after multiprocess writes: %v", err)
	}

	if len(facts) != processCount {
		t.Fatalf("expected %d durable facts written by %d competing processes (no fact lost), got %d: %+v",
			processCount, processCount, len(facts), facts)
	}

	seen := make(map[string]bool, processCount)
	for _, fct := range facts {
		seen[string(fct.Identity.Key)] = true
	}
	for i := 0; i < processCount; i++ {
		factID := fmt.Sprintf("proc-%d", i)
		catalog := durable.NewCatalog()
		key, keyErr := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "multiprocess-helper", Subject: factID})
		if keyErr != nil {
			t.Fatalf("derive expected semantic key for %s: %v", factID, keyErr)
		}
		if !seen[string(key)] {
			t.Errorf("fact from process %s missing from consolidated page — page corrupted or fact lost", factID)
		}
	}
}
