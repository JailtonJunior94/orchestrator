package procref_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/procref"
)

func TestCurrentProcessRefIsAlive(t *testing.T) {
	ref := procref.CurrentProcessRef()
	result := procref.DefaultLivenessProbe.Probe(ref)
	if !result.Reliable || !result.Alive {
		t.Fatalf("expected the current process to be reliably alive, got %+v", result)
	}
}

func TestExitedProcessIsNotAlive(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run throwaway process: %v", err)
	}
	pid := cmd.Process.Pid
	hostname, _ := os.Hostname()

	result := procref.DefaultLivenessProbe.Probe(procref.ProcessRef{PID: pid, Hostname: hostname})
	if !result.Reliable || result.Alive {
		t.Fatalf("expected exited process to be reliably dead, got %+v", result)
	}
}

func TestDifferentHostnameIsUnreliable(t *testing.T) {
	ref := procref.ProcessRef{PID: os.Getpid(), Hostname: "unreachable-host-xyz"}
	result := procref.DefaultLivenessProbe.Probe(ref)
	if result.Reliable {
		t.Fatalf("expected an unreachable hostname to be unreliable, got %+v", result)
	}
}

func TestNonPositivePIDIsReliablyDead(t *testing.T) {
	result := procref.DefaultLivenessProbe.Probe(procref.ProcessRef{PID: 0})
	if !result.Reliable || result.Alive {
		t.Fatalf("expected pid<=0 to be reliably dead, got %+v", result)
	}
}
