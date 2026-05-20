package persistence_test

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

// errFS é um FileSystem que falha em operações de escrita para testar caminhos de erro.
type errFS struct {
	*fs.FakeFileSystem
	writeErr error
	mkdirErr error
}

func (e *errFS) WriteFile(path string, data []byte) error {
	if e.writeErr != nil {
		return e.writeErr
	}
	return e.FakeFileSystem.WriteFile(path, data)
}

func (e *errFS) MkdirAll(path string) error {
	if e.mkdirErr != nil {
		return e.mkdirErr
	}
	return e.FakeFileSystem.MkdirAll(path)
}

func makeAgentMessageEvent(t *testing.T, text string) events.Event {
	t.Helper()
	evt, err := events.NewAgentMessage(time.Now(), text, json.RawMessage(`{"type":"agent_message"}`))
	if err != nil {
		t.Fatalf("NewAgentMessage: %v", err)
	}
	return evt
}

func TestNewJSONLWriter_InvalidPath(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	_, err := persistence.NewJSONLWriter("", fsys)
	if err == nil {
		t.Fatal("esperava erro para path vazio")
	}
}

func TestJSONLWriter_Append(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	w, err := persistence.NewJSONLWriter("/evidence/task-1/events.jsonl", fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter: %v", err)
	}

	msgs := []string{"olá mundo", "segunda mensagem", "terceira mensagem"}
	for _, msg := range msgs {
		evt := makeAgentMessageEvent(t, msg)
		if err := w.Append(evt); err != nil {
			t.Fatalf("Append(%q): %v", msg, err)
		}
	}

	data, err := fsys.ReadFile("/evidence/task-1/events.jsonl")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != len(msgs) {
		t.Fatalf("esperava %d linhas, obteve %d", len(msgs), len(lines))
	}

	for i, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("linha %d não é JSON válido: %v", i, err)
		}
	}
}

func TestJSONLWriter_AppendConcurrent(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	w, err := persistence.NewJSONLWriter("/evidence/task-2/events.jsonl", fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter: %v", err)
	}

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			text := strings.Repeat("x", i+1)
			evt := makeAgentMessageEvent(t, text)
			if err := w.Append(evt); err != nil {
				t.Errorf("goroutine %d: Append: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	data, err := fsys.ReadFile("/evidence/task-2/events.jsonl")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != n {
		t.Fatalf("esperava %d linhas, obteve %d", n, len(lines))
	}

	for i, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("linha %d não é JSON válido: %v", i, err)
		}
	}
}

func TestNewJSONLWriter_MkdirError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
		mkdirErr:       errors.New("disk full"),
	}
	_, err := persistence.NewJSONLWriter("/some/dir/events.jsonl", efs)
	if err == nil {
		t.Fatal("esperava erro de MkdirAll")
	}
}

func TestJSONLWriter_AppendWriteError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
	}
	w, err := persistence.NewJSONLWriter("/evidence/task-err/events.jsonl", efs)
	if err != nil {
		t.Fatalf("NewJSONLWriter: %v", err)
	}
	// Injetar falha de escrita após o writer ser criado.
	efs.writeErr = errors.New("write error")
	evt := makeAgentMessageEvent(t, "mensagem")
	if err := w.Append(evt); err == nil {
		t.Fatal("esperava erro de Append após falha de WriteFile")
	}
}

func TestJSONLWriter_ReusesExistingContent(t *testing.T) {
	fsys := fs.NewFakeFileSystem()

	// Criar writer e adicionar um evento.
	w1, err := persistence.NewJSONLWriter("/evidence/task-3/events.jsonl", fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter w1: %v", err)
	}
	if err := w1.Append(makeAgentMessageEvent(t, "primeiro")); err != nil {
		t.Fatalf("w1.Append: %v", err)
	}

	// Criar segundo writer para o mesmo arquivo (simula reinicialização).
	w2, err := persistence.NewJSONLWriter("/evidence/task-3/events.jsonl", fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter w2: %v", err)
	}
	if err := w2.Append(makeAgentMessageEvent(t, "segundo")); err != nil {
		t.Fatalf("w2.Append: %v", err)
	}

	data, _ := fsys.ReadFile("/evidence/task-3/events.jsonl")
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("esperava 2 linhas, obteve %d", len(lines))
	}
}
