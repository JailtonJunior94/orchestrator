package taskloop

import (
	"strings"
	"testing"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestRestoreTaskIsolationSnapshotAtRemovesUnexpectedTrackedFiles(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")

	snapshot, err := captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/task-3.0-intrusa.md"] = []byte("**Status:** pending\n")

	if err := restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
		t.Fatalf("restoreTaskIsolationSnapshotAt retornou erro inesperado: %v", err)
	}

	if _, err := fsys.ReadFile(prd + "/task-3.0-intrusa.md"); err == nil {
		t.Fatal("arquivo intruso deveria ter sido removido na restauracao")
	}
}

func TestValidateTaskFileIsolationRejectsUnexpectedTrackedFile(t *testing.T) {
	before := map[string][]byte{
		"/fake/project/tasks/prd-test/task-1.0-test.md": []byte("**Status:** pending\n"),
	}
	after := map[string][]byte{
		"/fake/project/tasks/prd-test/task-1.0-test.md":  []byte("**Status:** done\n"),
		"/fake/project/tasks/prd-test/task-2.0-extra.md": []byte("**Status:** pending\n"),
	}

	err := validateTaskFileIsolation(before, after, "/fake/project/tasks/prd-test/task-1.0-test.md", true)
	if err == nil {
		t.Fatal("esperado erro de arquivo novo, recebeu nil")
	}
	if !strings.Contains(err.Error(), "novo arquivo de task task-2.0-extra.md foi adicionado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsCurrentTaskMutation(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")

	snapshot, err := captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[currentTask] = []byte("**Status:** blocked\n")

	err = validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer altera a task atual")
	}
	if !strings.Contains(err.Error(), "arquivo de task task-1.0-test.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsCurrentTaskRowMutation(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")

	snapshot, err := captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | blocked | — | Nao |\n")

	err = validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer altera a row da task atual")
	}
	if !strings.Contains(err.Error(), "row da task 1.0 foi alterada indevidamente em tasks.md") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateProtectedPRDFileIsolationRejectsMutation(t *testing.T) {
	before := map[string][]byte{
		"/fake/project/tasks/prd-test/prd.md":      []byte("# PRD original\n"),
		"/fake/project/tasks/prd-test/techspec.md": []byte("# TechSpec original\n"),
	}
	after := map[string][]byte{
		"/fake/project/tasks/prd-test/prd.md":      []byte("# PRD alterado\n"),
		"/fake/project/tasks/prd-test/techspec.md": []byte("# TechSpec original\n"),
	}

	err := validateProtectedPRDFileIsolation(before, after)
	if err == nil {
		t.Fatal("esperado erro de mutacao em arquivo protegido do PRD, recebeu nil")
	}
	if !strings.Contains(err.Error(), "arquivo protegido do PRD prd.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestRestoreTaskIsolationSnapshotAtRestoresProtectedPRDFiles(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")

	snapshot, err := captureTaskIsolationSnapshot(prd, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshot retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/prd.md"] = []byte("# PRD alterado\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec alterado\n")

	if err := restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
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
	const prd = "/fake/project/tasks/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/notes.md"] = []byte("original\n")

	snapshot, err := captureTaskIsolationSnapshotWithMode(prd, taskIsolationModeExecutor, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/notes.md"] = []byte("alterado indevidamente\n")

	err = validateTaskIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando executor altera arquivo arbitrario do PRD")
	}
	if !strings.Contains(err.Error(), "arquivo protegido do PRD notes.md foi alterado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateReviewerIsolationRejectsArbitraryNestedPRDFileCreation(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"
	currentTask := prd + "/task-1.0-test.md"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[currentTask] = []byte("**Status:** done\n")
	fsys.Files[prd+"/docs/context.md"] = []byte("contexto\n")

	snapshot, err := captureTaskIsolationSnapshotWithMode(prd, taskIsolationModeReviewer, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/docs/review-notes.md"] = []byte("nao permitido\n")

	err = validateReviewerIsolation(snapshot, prd, "1.0", currentTask, fsys)
	if err == nil {
		t.Fatal("esperado erro quando reviewer cria arquivo arbitrario do PRD")
	}
	if !strings.Contains(err.Error(), "novo arquivo protegido do PRD review-notes.md foi adicionado indevidamente") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestRestoreTaskIsolationSnapshotAtRemovesUnexpectedProtectedPRDFiles(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/docs/context.md"] = []byte("contexto\n")

	snapshot, err := captureTaskIsolationSnapshotWithMode(prd, taskIsolationModeExecutor, fsys)
	if err != nil {
		t.Fatalf("captureTaskIsolationSnapshotWithMode retornou erro inesperado: %v", err)
	}

	fsys.Files[prd+"/docs/intruso.md"] = []byte("intruso\n")

	if err := restoreTaskIsolationSnapshotAt(snapshot, prd, fsys); err != nil {
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
			if got := isTrackedTaskFile(tt.filename); got != tt.want {
				t.Fatalf("isTrackedTaskFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
