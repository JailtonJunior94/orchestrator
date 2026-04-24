//go:build !windows

package taskloop

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunCmdStreamsStderrToLiveOut verifica que stderr do subprocesso e transmitido
// ao liveOut quando ele esta configurado (BUG-002 — ferramentas que escrevem em stderr
// agora ficam visiveis em tempo real, nao apenas capturadas no buffer do relatorio).
func TestRunCmdStreamsStderrToLiveOut(t *testing.T) {
	dir := t.TempDir()

	// Script que escreve em stderr e em stdout.
	script := "#!/bin/sh\nprintf 'stdout-line\n'\nprintf 'stderr-line\n' >&2\n"
	path := filepath.Join(dir, "dual-output")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar script: %v", err)
	}

	var liveCapture bytes.Buffer
	stdout, stderr, exitCode, err := runCmd(context.Background(), dir, &liveCapture, path)
	if err != nil {
		t.Fatalf("runCmd retornou erro inesperado: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exit code inesperado: %d", exitCode)
	}

	// stdout deve estar capturado no retorno
	if !strings.Contains(stdout, "stdout-line") {
		t.Errorf("stdout retornado deveria conter 'stdout-line', obteve: %q", stdout)
	}
	// stderr deve estar capturado no retorno
	if !strings.Contains(stderr, "stderr-line") {
		t.Errorf("stderr retornado deveria conter 'stderr-line', obteve: %q", stderr)
	}

	// liveOut deve ter recebido AMBOS stdout e stderr
	live := liveCapture.String()
	if !strings.Contains(live, "stdout-line") {
		t.Errorf("liveOut deveria conter stdout ('stdout-line'), obteve: %q", live)
	}
	if !strings.Contains(live, "stderr-line") {
		t.Errorf("liveOut deveria conter stderr ('stderr-line'), obteve: %q", live)
	}
}

// TestRunCmdNoLiveOutDoesNotPanic verifica que runCmd funciona normalmente
// sem liveOut configurado (nil) — nao afeta o caminho existente.
func TestRunCmdNoLiveOutDoesNotPanic(t *testing.T) {
	dir := t.TempDir()

	script := "#!/bin/sh\nprintf 'hello\n'\n"
	path := filepath.Join(dir, "hello")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar script: %v", err)
	}

	stdout, _, exitCode, err := runCmd(context.Background(), dir, nil, path)
	if err != nil {
		t.Fatalf("runCmd retornou erro: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exit code inesperado: %d", exitCode)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout deveria conter 'hello', obteve: %q", stdout)
	}
}

func TestRunCmdMonitoredCallsStartHook(t *testing.T) {
	dir := t.TempDir()

	script := "#!/bin/sh\nprintf 'hello\n'\n"
	path := filepath.Join(dir, "hello")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar script: %v", err)
	}

	startCalls := 0
	stdout, _, exitCode, err := runCmdMonitored(context.Background(), dir, nil, func() {
		startCalls++
	}, path)
	if err != nil {
		t.Fatalf("runCmdMonitored retornou erro: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exit code inesperado: %d", exitCode)
	}
	if startCalls != 1 {
		t.Fatalf("hook de start chamado %d vezes, want 1", startCalls)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout deveria conter 'hello', obteve: %q", stdout)
	}
}
