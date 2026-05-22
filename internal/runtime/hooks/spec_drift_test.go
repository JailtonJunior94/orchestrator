package hooks_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

// calcSHA256 calcula o hash SHA-256 hexadecimal do conteúdo fornecido.
func calcSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

// makeTasksDir cria um diretório temporário com prd.md (opcional) e tasks.md.
func makeTasksDir(t *testing.T, prdContent, tasksContent string) string {
	t.Helper()
	dir := t.TempDir()
	if prdContent != "" {
		if err := os.WriteFile(filepath.Join(dir, "prd.md"), []byte(prdContent), 0o644); err != nil {
			t.Fatalf("criar prd.md: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasksContent), 0o644); err != nil {
		t.Fatalf("criar tasks.md: %v", err)
	}
	return dir
}

func TestSpecDriftHook_WrongEventType(t *testing.T) {
	h := hooks.NewSpecDriftHook(false)
	// Evento de tipo errado: deve ser ignorado silenciosamente.
	err := h.Run(context.Background(), hooks.PromptBuildEvent{})
	if err != nil {
		t.Fatalf("esperado nil para evento de tipo errado, got %v", err)
	}
}

func TestSpecDriftHook_TasksDirEmpty_NoOp(t *testing.T) {
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  "/some/workdir",
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: "", // vazio = F1 ad-hoc
	}
	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("esperado no-op quando TasksDir vazio, got %v", err)
	}
}

func TestSpecDriftHook_SkipDriftGuard_NoOp(t *testing.T) {
	// Mesmo com TasksDir preenchido (sem tasks.md), SkipDriftGuard deve retornar nil.
	dir := t.TempDir()
	h := hooks.NewSpecDriftHook(true) // skipDriftGuard=true
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}
	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("esperado no-op com SkipDriftGuard=true, got %v", err)
	}
}

func TestSpecDriftHook_NoHashFound_ErrPRDUntracked(t *testing.T) {
	prdContent := "# PRD de teste\n\nRequisito RF-01\n"
	// tasks.md sem hash de PRD — deve retornar ErrPRDUntracked (RG-02).
	tasksContent := "# Tasks\n\n| # | Título | Status |\n|---|---|---|\n| 1.0 | Foo | pending |\n"

	dir := makeTasksDir(t, prdContent, tasksContent)
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("esperado ErrPRDUntracked, got nil")
	}
	if !errors.Is(err, hooks.ErrPRDUntracked) {
		t.Fatalf("esperado errors.Is(err, ErrPRDUntracked)=true, got %v", err)
	}
}

func TestSpecDriftHook_HashDiverged_ErrSpecDrift(t *testing.T) {
	prdContent := "# PRD de teste\n\nRequisito RF-01\n"
	// tasks.md com hash errado (drift) — deve retornar ErrSpecDrift (RG-01).
	tasksContent := "<!-- spec-hash-prd: 0000000000000000000000000000000000000000000000000000000000000000 -->\n# Tasks\n\n| # | Título | Status |\n|---|---|---|\n| 1.0 | Foo | pending |\n"

	dir := makeTasksDir(t, prdContent, tasksContent)
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("esperado ErrSpecDrift, got nil")
	}
	if !errors.Is(err, hooks.ErrSpecDrift) {
		t.Fatalf("esperado errors.Is(err, ErrSpecDrift)=true, got %v", err)
	}
}

func TestSpecDriftHook_HashMatch_Pass(t *testing.T) {
	// prdContent sem IDs RF-nn/REQ-nn para evitar que CheckCoverage falhe.
	// CheckDrift faz coverage + hash; se coverage falhar, report.Pass=false → ErrSpecDrift.
	prdContent := "# PRD de teste\n\nDescrição do produto sem requisitos numerados.\n"

	// Hash correto calculado via crypto/sha256 (garante paridade com specdrift.CheckHash).
	correctHash := calcSHA256(prdContent)
	tasksContent := fmt.Sprintf("<!-- spec-hash-prd: %s -->\n# Tasks\n\n| # | Título | Status |\n|---|---|---|\n| 1.0 | Foo | pending |\n", correctHash)

	dir := makeTasksDir(t, prdContent, tasksContent)
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("esperado nil (hash correto), got %v", err)
	}
}

func TestSpecDriftHook_TasksMissing_InfraError(t *testing.T) {
	// Sem tasks.md: deve retornar erro de infraestrutura (não ErrSpecDrift nem ErrPRDUntracked).
	dir := t.TempDir()
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("esperado erro de infraestrutura (tasks.md ausente), got nil")
	}
	if errors.Is(err, hooks.ErrSpecDrift) || errors.Is(err, hooks.ErrPRDUntracked) {
		t.Fatalf("esperado erro de infraestrutura, não sentinel: %v", err)
	}
}

func TestSpecDriftHook_NoPRDFile_NoHashCheck(t *testing.T) {
	// Sem prd.md: specdrift pula a verificação (spec file opcional).
	// tasks.md sem hash => NoHashFound não ocorre porque não há spec para verificar.
	// CheckDrift retorna report.Pass=true (sem spec, sem falha).
	tasksContent := "# Tasks\n\n| # | Título | Status |\n|---|---|---|\n| 1.0 | Foo | pending |\n"
	dir := makeTasksDir(t, "", tasksContent) // sem prd.md
	h := hooks.NewSpecDriftHook(false)
	evt := hooks.RuntimePreOpenEvent{
		WorkDir:  dir,
		SpecID:   "claude",
		Launcher: "claude",
		TasksDir: dir,
	}

	// Sem spec file: hook deve retornar nil (sem spec para verificar).
	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("esperado nil quando prd.md ausente, got %v", err)
	}
}
