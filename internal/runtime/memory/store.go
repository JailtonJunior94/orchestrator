package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Constantes de limite default para os dois tiers do memory store.
const (
	DirName                  = "memory"
	WorkflowFileName         = "MEMORY.md"
	DefaultWorkflowLineLimit = 150
	DefaultWorkflowByteLimit = 12 * 1024
	DefaultTaskLineLimit     = 200
	DefaultTaskByteLimit     = 16 * 1024
)

// Limits define os limites de compactação para cada tier.
type Limits struct {
	WorkflowLines int
	WorkflowBytes int
	TaskLines     int
	TaskBytes     int
}

// DefaultLimits retorna os limites padrão.
func (c *Catalog) DefaultLimits() Limits {
	return Limits{
		WorkflowLines: DefaultWorkflowLineLimit,
		WorkflowBytes: DefaultWorkflowByteLimit,
		TaskLines:     DefaultTaskLineLimit,
		TaskBytes:     DefaultTaskByteLimit,
	}
}

// FileState descreve o estado atual de um arquivo de memória.
type FileState struct {
	Path            string
	LineCount       int
	ByteCount       int
	NeedsCompaction bool
}

// Document encapsula o conteúdo e estado de um arquivo de memória.
type Document struct {
	FileState
	Content string
	Exists  bool
}

// WriteMode determina como o conteúdo é escrito no arquivo.
type WriteMode string

const (
	// WriteModeReplace sobrescreve o conteúdo existente.
	WriteModeReplace WriteMode = "replace"
	// WriteModeAppend concatena ao conteúdo existente sem trim.
	WriteModeAppend WriteMode = "append"
)

// Store define a interface do memory store 2-tier.
type Store interface {
	ReadWorkflow(ctx context.Context) (Document, error)
	ReadTask(ctx context.Context, taskFileName string) (Document, error)
	WriteWorkflow(ctx context.Context, content string, mode WriteMode) error
	WriteTask(ctx context.Context, taskFileName, content string, mode WriteMode) error
}

type store struct {
	tasksDir string
	limits   Limits
}

var _ Store = (*store)(nil)

// New cria um Store com o diretório de tasks e limites fornecidos.
func New(tasksDir string, limits Limits) Store {
	return &store{
		tasksDir: tasksDir,
		limits:   limits,
	}
}

// directory retorna o caminho do diretório de memória dentro do tasksDir.
func (s *store) directory() string {
	return filepath.Join(s.tasksDir, DirName)
}

// workflowPath retorna o caminho do arquivo de workflow.
func (s *store) workflowPath() string {
	return filepath.Join(s.directory(), WorkflowFileName)
}

// taskPath retorna o caminho seguro de um arquivo de task.
// filepath.Base é defensivo: previne traversal ../../../etc/passwd.
func (s *store) taskPath(taskFileName string) string {
	return filepath.Join(s.directory(), filepath.Base(taskFileName))
}

// ensureDir garante que o diretório de memória existe.
func (s *store) ensureDir() error {
	return os.MkdirAll(s.directory(), 0o755)
}

// countLines conta o número de bytes '\n' no conteúdo.
// Não usa strings.Split para evitar custo de memória dobrado.
func (c *Catalog) countLines(content string) int {
	count := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			count++
		}
	}
	return count
}

// readFile lê um arquivo e retorna Document. Retorna Exists:false sem erro quando ausente.
func (c *Catalog) readFile(path string, lineLimit, byteLimit int) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{
				FileState: FileState{Path: path},
				Exists:    false,
			}, nil
		}
		return Document{}, fmt.Errorf("memory: read %s: %w", path, err)
	}

	content := string(data)
	lineCount := NewCatalog().countLines(content)
	byteCount := len(content)
	needsCompaction := lineCount > lineLimit || byteCount > byteLimit

	return Document{
		FileState: FileState{
			Path:            path,
			LineCount:       lineCount,
			ByteCount:       byteCount,
			NeedsCompaction: needsCompaction,
		},
		Content: content,
		Exists:  true,
	}, nil
}

// writeFile escreve conteúdo em um arquivo com o modo especificado.
func (c *Catalog) writeFile(path, content string, mode WriteMode) error {
	switch mode {
	case WriteModeReplace:
		return os.WriteFile(path, []byte(content), 0o644)
	case WriteModeAppend:
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("memory: open %s: %w", path, err)
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return fmt.Errorf("memory: write %s: %w", path, err)
		}
		return nil
	default:
		return fmt.Errorf("memory: modo de escrita inválido: %q", mode)
	}
}

// ReadWorkflow lê o documento de workflow do PRD.
func (s *store) ReadWorkflow(_ context.Context) (Document, error) {
	return NewCatalog().readFile(s.workflowPath(), s.limits.WorkflowLines, s.limits.WorkflowBytes)
}

// ReadTask lê o documento de uma task específica.
func (s *store) ReadTask(_ context.Context, taskFileName string) (Document, error) {
	return NewCatalog().readFile(s.taskPath(taskFileName), s.limits.TaskLines, s.limits.TaskBytes)
}

// WriteWorkflow escreve o documento de workflow do PRD.
func (s *store) WriteWorkflow(_ context.Context, content string, mode WriteMode) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("memory: criar diretório: %w", err)
	}
	return NewCatalog().writeFile(s.workflowPath(), content, mode)
}

// WriteTask escreve o documento de uma task específica.
func (s *store) WriteTask(_ context.Context, taskFileName, content string, mode WriteMode) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("memory: criar diretório: %w", err)
	}
	return NewCatalog().writeFile(s.taskPath(taskFileName), content, mode)
}
