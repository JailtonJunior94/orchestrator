//go:build !windows

package durable_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestCheckProcessLiveness_CurrentProcessIsAlive(t *testing.T) {
	t.Parallel()

	ref := durable.CurrentProcessRef()
	result := durable.DefaultLivenessProbe.Probe(ref)
	if !result.Reliable {
		t.Fatal("expected reliable probe for current process on unix")
	}
	if !result.Alive {
		t.Fatal("expected current process to be alive")
	}
}

func TestCheckProcessLiveness_ExitedProcessIsNotAlive(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run helper process: %v", err)
	}
	pid := cmd.Process.Pid

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("failed to resolve hostname: %v", err)
	}

	result := durable.DefaultLivenessProbe.Probe(durable.ProcessRef{PID: pid, Hostname: hostname})
	if !result.Reliable {
		t.Fatal("expected reliable probe for exited process on unix")
	}
	if result.Alive {
		t.Fatal("expected exited process to be reported as not alive")
	}
}

func TestCheckProcessLiveness_DifferentHostnameIsUnreliable(t *testing.T) {
	t.Parallel()

	ref := durable.ProcessRef{PID: os.Getpid(), Hostname: "unreachable-host-xyz"}
	result := durable.DefaultLivenessProbe.Probe(ref)
	if result.Reliable {
		t.Fatal("expected unreliable probe across hostnames")
	}
}
