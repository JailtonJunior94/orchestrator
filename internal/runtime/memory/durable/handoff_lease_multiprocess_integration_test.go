//go:build integration

package durable_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

const (
	envHandoffHelperFlag     = "AISPEC_HANDOFF_HELPER"
	envHandoffHelperTasksDir = "AISPEC_HANDOFF_TASKS_DIR"
	envHandoffHelperClaimant = "AISPEC_HANDOFF_CLAIMANT"
	envHandoffHelperOwnerPID = "AISPEC_HANDOFF_OWNER_PID"
	handoffHelperTestSelect  = "-test.run=^TestHelperProcessClaimHandoff$"
)

func TestHelperProcessClaimHandoff(t *testing.T) {
	if os.Getenv(envHandoffHelperFlag) != "1" {
		t.Skip("helper process only, invoked via subprocess reinvocation")
	}

	tasksDir := os.Getenv(envHandoffHelperTasksDir)
	claimant := os.Getenv(envHandoffHelperClaimant)
	ownerPIDStr := os.Getenv(envHandoffHelperOwnerPID)
	if tasksDir == "" || claimant == "" || ownerPIDStr == "" {
		t.Fatal("multiprocess handoff helper: missing required env vars")
	}
	ownerPID, err := strconv.Atoi(ownerPIDStr)
	if err != nil {
		t.Fatalf("multiprocess handoff helper: invalid owner pid: %v", err)
	}

	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		t.Fatalf("multiprocess handoff helper: resolve hostname: %v", hostErr)
	}

	fsys := fs.NewOSFileSystem()
	store := durable.NewHandoffLeaseStore(fsys, durable.DefaultLayerLocker, tasksDir)
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
	ref := durable.ProcessRef{PID: ownerPID, Hostname: hostname, StartedAt: time.Now()}

	record, claimErr := store.Claim(scope, durable.DefaultLeasePolicy, durable.LeaseOwner(claimant), ref, 5*time.Minute, time.Now())
	if claimErr != nil {
		fmt.Printf("RESULT outcome=%s err=%v\n", record.Outcome, claimErr)
		return
	}
	fmt.Printf("RESULT outcome=%s err=<nil>\n", record.Outcome)
}

func TestTwoRealProcessesCompeteForSameHandoffLease(t *testing.T) {
	tasksDir := t.TempDir()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}

	leasePath, pathErr := durable.HandoffLeasePath(scope)
	if pathErr != nil {
		t.Fatalf("resolve handoff lease path: %v", pathErr)
	}
	fsys := fs.NewOSFileSystem()
	if err := fsys.MkdirAll(filepath.Dir(leasePath)); err != nil {
		t.Fatalf("pre-create memory directory so both subprocesses can open the lock file: %v", err)
	}

	ownerPID := os.Getpid()
	const processCount = 2
	type outcome struct {
		claimant string
		output   string
		err      error
	}
	results := make(chan outcome, processCount)
	for i := 0; i < processCount; i++ {
		claimant := fmt.Sprintf("pid:helper-%d", i)
		go func(id string) {
			cmd := exec.Command(os.Args[0], handoffHelperTestSelect) //nolint:gosec
			cmd.Env = append(os.Environ(),
				envHandoffHelperFlag+"=1",
				envHandoffHelperTasksDir+"="+tasksDir,
				envHandoffHelperClaimant+"="+id,
				envHandoffHelperOwnerPID+"="+strconv.Itoa(ownerPID),
			)
			output, runErr := cmd.CombinedOutput()
			results <- outcome{claimant: id, output: string(output), err: runErr}
		}(claimant)
	}

	grantedCount := 0
	refusedCount := 0
	for i := 0; i < processCount; i++ {
		res := <-results
		if res.err != nil {
			t.Fatalf("subprocess %s failed: %v\noutput:\n%s", res.claimant, res.err, res.output)
		}
		switch {
		case containsOutcome(res.output, "granted"):
			grantedCount++
		case containsOutcome(res.output, "refused"), containsOutcome(res.output, "transferred"):
			refusedCount++
		default:
			t.Fatalf("subprocess %s produced unexpected output: %s", res.claimant, res.output)
		}
	}

	if grantedCount != 1 {
		t.Fatalf("expected exactly one process to be granted the lease (mutual exclusion), got %d granted, %d refused/transferred", grantedCount, refusedCount)
	}
	if refusedCount != processCount-1 {
		t.Fatalf("expected the other %d process(es) to be refused or detect the live owner, got %d", processCount-1, refusedCount)
	}

	store := durable.NewHandoffLeaseStore(fsys, durable.DefaultLayerLocker, tasksDir)
	lease, loadErr := store.Load(scope)
	if loadErr != nil {
		t.Fatalf("load persisted lease after multiprocess claim race: %v", loadErr)
	}
	if lease == nil {
		t.Fatal("expected a persisted lease after multiprocess claim race, got none")
	}
}

func containsOutcome(output, want string) bool {
	return strings.Contains(output, "outcome="+want)
}
