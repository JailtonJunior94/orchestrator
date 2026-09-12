//go:build integration

package aispecharness

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestMemoryMigrateRealFilesystemBackupVerifiedAndRefusesReapplication(t *testing.T) {
	tasksDir := t.TempDir()
	memoryDir := filepath.Join(tasksDir, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("MkdirAll memory dir: %v", err)
	}

	legacyContent := "# Legacy PRD Memory\n\nContexto acumulado manualmente antes da fatia 9.0.\n"
	legacyPath := filepath.Join(memoryDir, "MEMORY.md")
	if err := os.WriteFile(legacyPath, []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("write legacy MEMORY.md: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	handler := &memoryCommand{}
	printer, stdout, _ := newTestPrinter()

	if err := handler.runMigrate(printer, fsys, tasksDir); err != nil {
		t.Fatalf("runMigrate: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("runMigrate produced no output confirming the migration")
	}

	backupPath := legacyPath + migrationBackupSuffix
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup file %s: %v", backupPath, err)
	}
	if string(backupContent) != legacyContent {
		t.Fatalf("backup content does not match original byte for byte:\nwant: %q\ngot:  %q", legacyContent, string(backupContent))
	}

	migratedContent, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read migrated MEMORY.md: %v", err)
	}

	page := durable.NewMarkdownPage()
	facts, human, parseErr := page.Parse(migratedContent)
	if parseErr != nil {
		t.Fatalf("parse migrated page: %v", parseErr)
	}
	if len(facts) != 0 {
		t.Fatalf("migrated page must carry zero Facts (legacy content is preserved only as HumanBlock), got %d", len(facts))
	}
	if human.Header.FormatVersion != durable.FormatVersionCurrent {
		t.Fatalf("migrated page format_version = %d, want %d", human.Header.FormatVersion, durable.FormatVersionCurrent)
	}
	if human.Content != legacyContent {
		t.Fatalf("migrated human content not preserved:\nwant: %q\ngot:  %q", legacyContent, human.Content)
	}

	reapplyErr := handler.runMigrate(printer, fsys, tasksDir)
	if reapplyErr == nil {
		t.Fatal("reapplying migration over an already-converted page must be refused, got nil error")
	}
	if !errors.Is(reapplyErr, durable.ErrMigrationAlreadyApplied) {
		t.Fatalf("reapplication error = %v, want wrapping durable.ErrMigrationAlreadyApplied", reapplyErr)
	}

	afterRefusal, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read page after refused reapplication: %v", err)
	}
	if string(afterRefusal) != string(migratedContent) {
		t.Fatal("refused reapplication must not modify the already-migrated page on disk")
	}
}
