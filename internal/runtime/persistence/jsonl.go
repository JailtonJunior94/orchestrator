// Package persistence implementa a camada de IO do runtime ACP.
// Responsabilidades: writer JSONL append-only, renderer de tool_calls.md e
// enriquecimento idempotente do execution_report.md.
package persistence

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// JSONLWriter escreve eventos em um arquivo JSONL append-only (RF-08).
// É seguro para uso concorrente por múltiplas goroutines.
type JSONLWriter struct {
	path string
	fsys fs.FileSystem
	mu   sync.Mutex
	// buf acumula o conteúdo do arquivo em memória (necessário pois FakeFileSystem
	// não suporta append; OSFileSystem recebe dados completos via WriteFile).
	buf []byte
}

// NewJSONLWriter cria um JSONLWriter validando o path e criando o diretório pai.
// Retorna erro se o path estiver vazio após normalização.
func NewJSONLWriter(path string, fsys fs.FileSystem) (*JSONLWriter, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return nil, fmt.Errorf("persistence: path inválido para JSONLWriter: %q", path)
	}
	dir := filepath.Dir(clean)
	if err := fsys.MkdirAll(dir); err != nil {
		return nil, fmt.Errorf("persistence: criar diretório %s: %w", dir, err)
	}
	// Carregar conteúdo existente, se houver.
	existing, err := fsys.ReadFile(clean)
	if err == nil {
		return &JSONLWriter{path: clean, fsys: fsys, buf: existing}, nil
	}
	return &JSONLWriter{path: clean, fsys: fsys}, nil
}

// Append serializa evt e anexa uma linha ao arquivo JSONL.
// Falha de escrita propaga erro sem corromper o conteúdo anterior.
func (w *JSONLWriter) Append(evt events.Event) error {
	data, err := evt.MarshalJSON()
	if err != nil {
		return fmt.Errorf("appending event to %s: %w", w.path, err)
	}
	line := append(data, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	newBuf := append(w.buf, line...)
	if err := w.fsys.WriteFile(w.path, newBuf); err != nil {
		return fmt.Errorf("appending event to %s: %w", w.path, err)
	}
	w.buf = newBuf
	return nil
}
