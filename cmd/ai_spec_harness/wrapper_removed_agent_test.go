package aispecharness

import (
	"os"
	"strings"
	"testing"
)

type stderrCapture struct{}

func (c stderrCapture) around(t *testing.T, run func()) string {
	t.Helper()
	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = writer
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 8192)
		n, _ := reader.Read(buf)
		done <- string(buf[:n])
	}()
	run()
	_ = writer.Close()
	os.Stderr = original
	return <-done
}

func TestWrapperCmd_RemovedAgentGemini(t *testing.T) {
	cmd := newWrapperCmd()
	cmd.SetArgs([]string{"gemini", "go-implementation", "."})
	var err error
	out := stderrCapture{}.around(t, func() { err = cmd.Execute() })

	if err == nil {
		t.Fatal("esperava erro de saida para agente removido")
	}
	if strings.Contains(out, "ferramenta invalida \"gemini\"") {
		t.Errorf("RF-03: erro generico de valor invalido nao deve ser usado para gemini: %s", out)
	}
	if !strings.Contains(out, "migracao") && !strings.Contains(out, "migra") {
		t.Errorf("erro deveria apontar o guia de migracao: %s", out)
	}
}
