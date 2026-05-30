package memory_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
)

// T-MEM-01: Ciclo completo write+read no tier workflow.
func TestTMEM01_WriteReadWorkflow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits()
	s := memory.New(dir, limits)
	ctx := context.Background()

	content := "# Workflow Memory\nLinha 1\nLinha 2\n"

	if err := s.WriteWorkflow(ctx, content, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteWorkflow: %v", err)
	}

	doc, err := s.ReadWorkflow(ctx)
	if err != nil {
		t.Fatalf("ReadWorkflow: %v", err)
	}

	if !doc.Exists {
		t.Fatal("esperado Exists=true após escrita")
	}
	if doc.Content != content {
		t.Fatalf("conteúdo divergente: got=%q want=%q", doc.Content, content)
	}
	if doc.NeedsCompaction {
		t.Fatal("NeedsCompaction não deveria estar setado para conteúdo pequeno")
	}
}

// T-MEM-02: 151 linhas → NeedsCompaction=true (limite de linhas de workflow).
func TestTMEM02_WorkflowLineCompaction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits() // WorkflowLines = 150
	s := memory.New(dir, limits)
	ctx := context.Background()

	// Construir conteúdo com exatamente 151 linhas (151 '\n').
	var sb strings.Builder
	for i := 0; i < 151; i++ {
		sb.WriteString("linha\n")
	}
	content := sb.String()

	if err := s.WriteWorkflow(ctx, content, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteWorkflow: %v", err)
	}

	doc, err := s.ReadWorkflow(ctx)
	if err != nil {
		t.Fatalf("ReadWorkflow: %v", err)
	}

	if !doc.NeedsCompaction {
		t.Fatalf("NeedsCompaction deveria ser true com %d linhas (limite=%d)", doc.LineCount, limits.WorkflowLines)
	}
	if doc.LineCount != 151 {
		t.Fatalf("LineCount esperado=151 got=%d", doc.LineCount)
	}
}

// T-MEM-03: 50 linhas com 300 bytes cada (15000 bytes total > 12288 limite) → NeedsCompaction=true.
func TestTMEM03_WorkflowByteCompaction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits() // WorkflowBytes = 12*1024 = 12288
	s := memory.New(dir, limits)
	ctx := context.Background()

	// Construir 50 linhas de 299 chars + '\n' = 300 bytes cada = 15000 bytes total.
	line := strings.Repeat("x", 299) + "\n"
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString(line)
	}
	content := sb.String()

	// Verificar tamanho esperado.
	if len(content) != 15000 {
		t.Fatalf("setup: tamanho esperado=15000 got=%d", len(content))
	}

	if err := s.WriteWorkflow(ctx, content, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteWorkflow: %v", err)
	}

	doc, err := s.ReadWorkflow(ctx)
	if err != nil {
		t.Fatalf("ReadWorkflow: %v", err)
	}

	if !doc.NeedsCompaction {
		t.Fatalf("NeedsCompaction deveria ser true: bytes=%d > limite=%d", doc.ByteCount, limits.WorkflowBytes)
	}
	if doc.ByteCount != 15000 {
		t.Fatalf("ByteCount esperado=15000 got=%d", doc.ByteCount)
	}
	// 50 linhas: 50 '\n', portanto LineCount=50, abaixo do limite de 150.
	if doc.LineCount != 50 {
		t.Fatalf("LineCount esperado=50 got=%d", doc.LineCount)
	}
}

// T-MEM-04: Dir sem MEMORY.md → Document{Exists:false}, err=nil.
func TestTMEM04_ReadWorkflowNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits()
	s := memory.New(dir, limits)
	ctx := context.Background()

	doc, err := s.ReadWorkflow(ctx)
	if err != nil {
		t.Fatalf("ReadWorkflow deveria retornar nil err quando arquivo ausente, got: %v", err)
	}
	if doc.Exists {
		t.Fatal("Exists deveria ser false quando arquivo não existe")
	}
	if doc.Content != "" {
		t.Fatalf("Content deveria ser vazio, got=%q", doc.Content)
	}
}

// T-MEM-05: Append mode preserva conteúdo prévio sem trim.
func TestTMEM05_AppendMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits()
	s := memory.New(dir, limits)
	ctx := context.Background()

	first := "# Sessão 1\nresultado anterior\n"
	second := "# Sessão 2\nnovo resultado\n"

	if err := s.WriteWorkflow(ctx, first, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteWorkflow replace: %v", err)
	}
	if err := s.WriteWorkflow(ctx, second, memory.WriteModeAppend); err != nil {
		t.Fatalf("WriteWorkflow append: %v", err)
	}

	doc, err := s.ReadWorkflow(ctx)
	if err != nil {
		t.Fatalf("ReadWorkflow: %v", err)
	}

	expected := first + second
	if doc.Content != expected {
		t.Fatalf("append não preservou conteúdo:\ngot=%q\nwant=%q", doc.Content, expected)
	}
}

// Testes adicionais para cobrir o tier de task e paths seguros.

func TestReadTaskNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := memory.New(dir, memory.NewCatalog().DefaultLimits())
	ctx := context.Background()

	doc, err := s.ReadTask(ctx, "task-inexistente.md")
	if err != nil {
		t.Fatalf("ReadTask deveria retornar nil err, got: %v", err)
	}
	if doc.Exists {
		t.Fatal("Exists deveria ser false")
	}
}

func TestWriteReadTask(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := memory.New(dir, memory.NewCatalog().DefaultLimits())
	ctx := context.Background()

	content := "# Task Memory\ncontexto relevante\n"
	taskFile := "task-1.0.md"

	if err := s.WriteTask(ctx, taskFile, content, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteTask: %v", err)
	}

	doc, err := s.ReadTask(ctx, taskFile)
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}

	if !doc.Exists {
		t.Fatal("Exists deveria ser true após escrita")
	}
	if doc.Content != content {
		t.Fatalf("content divergente: got=%q want=%q", doc.Content, content)
	}
}

func TestTaskLineCompaction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	limits := memory.NewCatalog().DefaultLimits() // TaskLines = 200
	s := memory.New(dir, limits)
	ctx := context.Background()

	var sb strings.Builder
	for i := 0; i < 201; i++ {
		sb.WriteString("x\n")
	}

	if err := s.WriteTask(ctx, "task.md", sb.String(), memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteTask: %v", err)
	}

	doc, err := s.ReadTask(ctx, "task.md")
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if !doc.NeedsCompaction {
		t.Fatalf("NeedsCompaction deveria ser true com %d linhas", doc.LineCount)
	}
}

func TestTaskAppendMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := memory.New(dir, memory.NewCatalog().DefaultLimits())
	ctx := context.Background()

	first := "parte 1\n"
	second := "parte 2\n"

	if err := s.WriteTask(ctx, "task.md", first, memory.WriteModeReplace); err != nil {
		t.Fatalf("WriteTask replace: %v", err)
	}
	if err := s.WriteTask(ctx, "task.md", second, memory.WriteModeAppend); err != nil {
		t.Fatalf("WriteTask append: %v", err)
	}

	doc, err := s.ReadTask(ctx, "task.md")
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if doc.Content != first+second {
		t.Fatalf("append incorreto: got=%q", doc.Content)
	}
}

func TestTaskPathTraversalPrevention(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := memory.New(dir, memory.NewCatalog().DefaultLimits())
	ctx := context.Background()

	// Tentar traversal: filepath.Base deve prevenir isso.
	// "../../../etc/passwd" → Base = "passwd" → gravado dentro do dir de memória.
	err := s.WriteTask(ctx, "../../../etc/passwd", "conteudo", memory.WriteModeReplace)
	if err != nil {
		t.Fatalf("WriteTask: %v", err)
	}

	doc, err := s.ReadTask(ctx, "../../../etc/passwd")
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}

	// O arquivo deve estar dentro do diretório de memória do store, não em /etc/.
	expectedPath := filepath.Join(dir, "memory", "passwd")
	if doc.Path != expectedPath {
		t.Fatalf("path de traversal não prevenido:\ngot=%q\nwant=%q", doc.Path, expectedPath)
	}
}

func TestWriteModeInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := memory.New(dir, memory.NewCatalog().DefaultLimits())
	ctx := context.Background()

	// Primeiro garantir que o diretório e arquivo existem.
	if err := s.WriteWorkflow(ctx, "seed\n", memory.WriteModeReplace); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := s.WriteWorkflow(ctx, "conteudo", "invalido")
	if err == nil {
		t.Fatal("esperado erro para WriteMode inválido")
	}
}

func TestDefaultLimits(t *testing.T) {
	t.Parallel()

	l := memory.NewCatalog().DefaultLimits()
	if l.WorkflowLines != memory.DefaultWorkflowLineLimit {
		t.Errorf("WorkflowLines: got=%d want=%d", l.WorkflowLines, memory.DefaultWorkflowLineLimit)
	}
	if l.WorkflowBytes != memory.DefaultWorkflowByteLimit {
		t.Errorf("WorkflowBytes: got=%d want=%d", l.WorkflowBytes, memory.DefaultWorkflowByteLimit)
	}
	if l.TaskLines != memory.DefaultTaskLineLimit {
		t.Errorf("TaskLines: got=%d want=%d", l.TaskLines, memory.DefaultTaskLineLimit)
	}
	if l.TaskBytes != memory.DefaultTaskByteLimit {
		t.Errorf("TaskBytes: got=%d want=%d", l.TaskBytes, memory.DefaultTaskByteLimit)
	}
}
