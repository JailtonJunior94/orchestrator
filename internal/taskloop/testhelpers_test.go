package taskloop

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

// captureStderr redireciona os.Stderr durante f e retorna tudo que foi escrito.
// Helper compartilhado entre testes unitarios e de integracao.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	original := os.Stderr
	os.Stderr = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, r)
	}()

	f()

	_ = w.Close()
	os.Stderr = original
	wg.Wait()
	_ = r.Close()
	return buf.String()
}
