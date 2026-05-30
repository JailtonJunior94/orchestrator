package taskloop

import (
	"strings"
	"testing"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestRestoreTaskIsolationSnapshotAtRemovesUnexpectedTrackedFiles(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/task-3.0-intrusa.md"] = []byte("**Status:** pending\n")

	if err := NewCatalog().restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
		t.Fatalf("restoreTaskIsolationSnapshotAt retornou erro inesperado: %v", err)
	}

	if _, err := fsys.ReadFile(prd + "/task-3.0-intrusa.md"); err == nil {
		t.Fatal("arquivo intruso deveria ter sido removido na restauracao")
	}
}

func TestValidateTaskFileIsolationRejectsUnexpectedTrackedFile(t *testing.T) {
	before := map[string][]byte{
		"/fake/project/.specs/prd-test/task-1.0-test.md": []byte("**Status:** pending\n"),
	}
	after := map[string][]byte{
		"/fake/project/.specs/prd-test/task-1.0-test.md":  []byte("**Status:** done\n"),
		"/fake/project/.specs/prd-test/task-2.0-extra.md": []byte("**Status:** pending\n"),
	}

	err := NewCatalog().validateTaskFileIsolation(before, after, "/fake/project/.specs/prd-test/task-1.0-test.md", true)
	if err == nil {
		t.Fatal("esperado erro de arquivo novo, recebeu nil")
	}
	if !strings.Contains(err.Error(), "novo arquivo de task task-2.0-extra.md foi adicionado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsCurrentTaskMutation(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[currentTask] = []byte("**Status:** blocked\n")

	err = NewCatalog().validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer altera a task atual")
	}
	if !strings.Contains(err.Error(), "arquivo de task task-1.0-test.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsCurrentTaskRowMutation(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | blocked | — | Nao |\n")

	err = NewCatalog().validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer altera a row da task atual")
	}
	if !strings.Contains(err.Error(), "row da task 1.0 foi alterada indevidamente em tasks.md") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateProtectedPRDFileIsolationRejectsMutation(t *testing.T) {
	before := map[string][]byte{
		"/fake/project/.specs/prd-test/prd.md":      []byte("# PRD original\n"),
		"/fake/project/.specs/prd-test/techspec.md": []byte("# TechSpec original\n"),
	}
	after := map[string][]byte{
		"/fake/project/.specs/prd-test/prd.md":      []byte("# PRD alterado\n"),
		"/fake/project/.specs/prd-test/techspec.md": []byte("# TechSpec original\n"),
	}

	err := NewCatalog().validateProtectedPRDFileIsolation(before, after)
	if err == nil {
		t.Fatal("esperado erro de mutacao em arquivo protegido do PRD, recebeu nil")
	}
	if !strings.Contains(err.Error(), "arquivo protegido do PRD prd.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestRestoreTaskIsolationSnapshotAtRestoresProtectedPRDFiles(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/prd.md"] = []byte("# PRD alterado\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec alterado\n")

	if err := NewCatalog().restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
		t.Fatalf("restoreTaskIsolationSnapshotAt retornou erro inesperado: %v", err)
	}

	if got := string(fsys.Files[prd+"/prd.md"]); got != "# PRD original\n" {
		t.Fatalf("prd.md deveria ter sido restaurado, obteve: %q", got)
	}
	if got := string(fsys.Files[prd+"/techspec.md"]); got != "# TechSpec original\n" {
		t.Fatalf("techspec.md deveria ter sido restaurado, obteve: %q", got)
	}
}

func TestValidateTaskIsolationRejectsArbitraryPRDFileMutation(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/notes.md"] = []byte("original\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshotWithMode(prd, _taskIsolationModeExecutor, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/notes.md"] = []byte("alterado indevidamente\n")

	err = NewCatalog().validateTaskIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando executor altera arquivo arbitrario do PRD")
	}
	if !strings.Contains(err.Error(), "arquivo protegido do PRD notes.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsArbitraryNestedPRDFileCreation(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")
	fsys.Files[prd+"/docs/context.md"] = []byte("contexto\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshotWithMode(prd, _taskIsolationModeReviewer, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/docs/review-notes.md"] = []byte("nao permitido\n")

	err = NewCatalog().validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer cria arquivo arbitrario do PRD")
	}
	if !strings.Contains(err.Error(), "novo arquivo protegido do PRD review-notes.md foi adicionado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

// TestValidateTaskIsolationAllowsMemoryDir é a regressão do conflito memory_persist × isolamento:
// o hook memory_persist grava .specs/<prd>/memory/MEMORY.md APÓS a sessão; isso NÃO deve disparar
// violação de isolamento (o subdir memory/ é gerenciado pelo harness, não pelo agente).
func TestValidateTaskIsolationAllowsMemoryDir(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** pending\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	// Hook memory_persist grava MEMORY.md no subdir memory/ após a sessão.
	fsys.Files[prd+"/memory/MEMORY.md"] = []byte("# Memory\n")
	fsys.Files[prd+"/1.0_execution_report.md"] = []byte("# Generated\n")

	if err := NewCatalog().validateTaskIsolation(snapshot, prd, "1.0", currentTask, fsys); err != nil {
		t.Fatalf("memory/MEMORY.md (hook memory_persist) não deve disparar violação de isolamento: %v", err)
	}
}

// TestIsProtectedPRDFile_HarnessManagedDirsExcluded valida que os subdirs gerenciados pela stack
// (memory/, .checkpoints/, .partials/) não são protegidos — artefatos da própria stack não devem
// disparar violação de isolamento (regressão dos falsos-positivos MEMORY.md e .checkpoints/<n>.yaml).
func TestIsProtectedPRDFile_HarnessManagedDirsExcluded(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"
	managed := []string{
		prd + "/memory/MEMORY.md",
		prd + "/.checkpoints/1.0.yaml",
		prd + "/.partials/tasks.md.1.0.partial",
	}
	for _, mode := range []taskIsolationMode{_taskIsolationModeExecutor, _taskIsolationModeReviewer} {
		for _, p := range managed {
			if NewCatalog().isProtectedPRDFile(prd, p, mode) {
				t.Errorf("%s não deveria ser protegido (gerenciado pela stack, mode=%d)", p, mode)
			}
		}
	}
	// Arquivo arbitrário no PRD folder continua protegido.
	if !NewCatalog().isProtectedPRDFile(prd, prd+"/adr-001.md", _taskIsolationModeReviewer) {
		t.Error("adr-001.md deveria continuar protegido")
	}
}

func TestRestoreTaskIsolationSnapshotAtRemovesUnexpectedProtectedPRDFiles(t *testing.T) {
	const prd = "/fake/project/.specs/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/docs/context.md"] = []byte("contexto\n")

	snapshot, err := NewCatalog().captureTaskIsolationSnapshotWithMode(prd, _taskIsolationModeExecutor, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/docs/intruso.md"] = []byte("intruso\n")

	if err := NewCatalog().restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
		t.Fatalf("restoreTaskIsolationSnapshotAt retornou erro inesperado: %v", err)
	}

	if _, err := fsys.ReadFile(prd + "/docs/intruso.md"); err == nil {
		t.Fatal("arquivo protegido intruso deveria ter sido removido na restauracao")
	}
}

func TestIsTrackedTaskFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "task com prefixo simples", filename: "1_task.md", want: true},
		{name: "task com id completo", filename: "1.0-desc.md", want: true},
		{name: "task com prefixo task", filename: "task-1.0-test.md", want: true},
		{name: "task zero padded", filename: "TASK-001-desc.md", want: true},
		{name: "arquivo arbitrario do prd", filename: "notes.md", want: false},
		{name: "arquivo protegido prd", filename: "prd.md", want: false},
		{name: "execution report", filename: "1.0_execution_report.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCatalog().isTrackedTaskFile(tt.filename); got != tt.want {
				t.Fatalf("isTrackedTaskFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
