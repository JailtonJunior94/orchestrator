package persistence

import (
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// SessionPersistence implementa runtime.Persistence para uma sessão ACP.
// Criado por SessionPersistenceFactory e injetado no ACPRunner.
type SessionPersistence struct {
	jsonl       *JSONLWriter
	evidenceDir string
	fsys        fs.FileSystem
}

var _ airuntime.Persistence = (*SessionPersistence)(nil)

// NewSessionPersistence cria um SessionPersistence para o evidenceDir fornecido.
func NewSessionPersistence(evidenceDir string, fsys fs.FileSystem) (*SessionPersistence, error) {
	eventsPath := filepath.Join(evidenceDir, "events.jsonl")
	jsonl, err := NewJSONLWriter(eventsPath, fsys)
	if err != nil {
		return nil, err
	}
	return &SessionPersistence{
		jsonl:       jsonl,
		evidenceDir: evidenceDir,
		fsys:        fsys,
	}, nil
}

// AppendEvent persiste um evento no arquivo events.jsonl.
func (s *SessionPersistence) AppendEvent(evt events.Event) error {
	return s.jsonl.Append(evt)
}

// WriteToolCalls gera o arquivo tool_calls.md.
func (s *SessionPersistence) WriteToolCalls(summaries []events.ToolCallSummary) error {
	path := filepath.Join(s.evidenceDir, "tool_calls.md")
	return NewCatalog().WriteToolCalls(path, summaries, s.fsys)
}

// EnrichReport enriquece o execution_report.md com o summary da sessão.
func (s *SessionPersistence) EnrichReport(summary airuntime.Summary) error {
	reportPath := filepath.Join(s.evidenceDir, "execution_report.md")
	return NewCatalog().EnrichReport(reportPath, summary, s.fsys)
}

// sessionPersistenceFactory implementa runtime.PersistenceFactory.
type sessionPersistenceFactory struct {
	fsys fs.FileSystem
}

var _ airuntime.PersistenceFactory = (*sessionPersistenceFactory)(nil)

// NewSessionPersistenceFactory cria uma PersistenceFactory que usa o FileSystem fornecido.
func NewSessionPersistenceFactory(fsys fs.FileSystem) airuntime.PersistenceFactory {
	return &sessionPersistenceFactory{fsys: fsys}
}

// New cria uma SessionPersistence para o evidenceDir fornecido.
func (f *sessionPersistenceFactory) New(evidenceDir string) (airuntime.Persistence, error) {
	return NewSessionPersistence(evidenceDir, f.fsys)
}
