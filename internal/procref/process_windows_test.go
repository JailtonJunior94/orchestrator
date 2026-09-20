//go:build windows

package procref_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/procref"
)

func TestWindowsCurrentProcessIsAlive(t *testing.T) {
	ref := procref.CurrentProcessRef()
	result := procref.DefaultLivenessProbe.Probe(ref)
	if !result.Reliable || !result.Alive {
		t.Fatalf("expected the current process to be reliably alive, got %+v", result)
	}
}

func TestWindowsExitedProcessIsNotAlive(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit", "0")
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
