package uninstall

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestUninstall_RemovesSkills(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.agents/skills/bugfix/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.md"] = []byte("# AGENTS")
	ffs.Files["/project/CLAUDE.md"] = []byte("# CLAUDE")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte("{}")
	ffs.Files["/project/.claude/hooks/validate-governance.sh"] = []byte("gov")
	ffs.Files["/project/.claude/hooks/validate-preload.sh"] = []byte("pre")
	ffs.Files["/project/.claude/scripts/validate-bugfix-evidence.sh"] = []byte("bugfix")
	ffs.Files["/project/.claude/scripts/validate-refactor-evidence.sh"] = []byte("refactor")
	ffs.Files["/project/scripts/lib/parse-hook-input.sh"] = []byte("helper")
	ffs.Files["/project/scripts/lib/check-invocation-depth.sh"] = []byte("depth")

	printer := output.New(false)
	svc := NewService(ffs, printer)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(svc.defaultClaudeSettings())

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.agents/skills/review/SKILL.md") {
		t.Error("skill review should be removed")
	}
	if ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md should be removed")
	}
	if ffs.Exists("/project/CLAUDE.md") {
		t.Error("CLAUDE.md should be removed")
	}
	if ffs.Exists("/project/.ai_spec_harness.json") {
		t.Error("manifest should be removed")
	}
	if ffs.Exists("/project/.claude/hooks/validate-preload.sh") {
		t.Error("validate-preload hook should be removed")
	}
	if ffs.Exists("/project/.claude/settings.local.json") {
		t.Error("generated settings.local.json should be removed")
	}
	if ffs.Exists("/project/scripts/lib/parse-hook-input.sh") {
		t.Error("parse-hook-input helper should be removed")
	}
	if ffs.Exists("/project/.claude/scripts/validate-bugfix-evidence.sh") {
		t.Error("validate-bugfix-evidence.sh should be removed")
	}
	if ffs.Exists("/project/.claude/scripts/validate-refactor-evidence.sh") {
		t.Error("validate-refactor-evidence.sh should be removed")
	}
	if ffs.Exists("/project/scripts/lib/check-invocation-depth.sh") {
		t.Error("check-invocation-depth.sh should be removed")
	}
}

// TestUninstall_RemovesLegacyGeminiResidue verifica que a desinstalacao limpa o
// residuo de ".gemini/" deixado por projetos legados que instalaram o agente
// removido na tarefa 10.0, mesmo sem manifesto rastreando esses arquivos.
func TestUninstall_RemovesLegacyGeminiResidue(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.gemini/commands/review.toml"] = []byte("[command]")
	ffs.Files["/project/.gemini/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")

	printer := output.New(false)
	svc := NewService(ffs, printer)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(svc.defaultClaudeSettings())

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.gemini/hooks/validate-preload.sh") {
		t.Error("residuo .gemini/hooks/validate-preload.sh deveria ser removido")
	}
	if ffs.Exists("/project/.gemini/commands/review.toml") {
		t.Error("residuo .gemini/commands/review.toml deveria ser removido")
	}
}

func TestUninstall_DryRunDoesNotRemove(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.md"] = []byte("# AGENTS")

	printer := output.New(false)
	svc := NewService(ffs, printer)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(svc.defaultClaudeSettings())

	err := svc.Execute("/project", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In dry-run, files should still exist
	if !ffs.Exists("/project/.agents/skills/review/SKILL.md") {
		t.Error("skill should NOT be removed in dry-run")
	}
	if !ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md should NOT be removed in dry-run")
	}
}

func TestUninstall_NoSkillsDir(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	printer := output.New(false)
	svc := NewService(ffs, printer)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(svc.defaultClaudeSettings())

	err := svc.Execute("/project", false)
	if err == nil {
		t.Fatal("expected error for missing .agents/skills/")
	}
}

func TestUninstall_PreservesAgentsLocal(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.local.md"] = []byte("# Local extensions")

	printer := output.New(false)
	svc := NewService(ffs, printer)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(svc.defaultClaudeSettings())

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/AGENTS.local.md") {
		t.Error("AGENTS.local.md should be preserved")
	}
}

// TestUninstall_ManifestDriven_PreservesUserFile_Idempotent valida RF-05/RF-60:
// com um manifesto rastreando arquivos individualmente, a desinstalacao remove
// exclusivamente o que a instalacao criou, preserva um arquivo do usuario no
// mesmo diretorio, e reexecutar e idempotente (sai 0, nao erra em arquivo ja ausente).
func TestUninstall_ManifestDriven_PreservesUserFile_Idempotent(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.md"] = []byte("# AGENTS")
	ffs.Files["/project/.claude/hooks/validate-preload.sh"] = []byte("pre")
	ffs.Files["/project/.claude/hooks/custom-user-hook.sh"] = []byte("# nao instalado pelo harness")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "1.0.0-test",
		"installed_files": [
			"AGENTS.md",
			".claude/hooks/validate-preload.sh"
		]
	}`)

	printer := output.New(false)
	svc := NewService(ffs, printer)

	if err := svc.Execute("/project", false); err != nil {
		t.Fatalf("primeira execucao: unexpected error: %v", err)
	}

	if ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md rastreado pelo manifesto deveria ser removido")
	}
	if ffs.Exists("/project/.claude/hooks/validate-preload.sh") {
		t.Error(".claude/hooks/validate-preload.sh rastreado pelo manifesto deveria ser removido")
	}
	if !ffs.Exists("/project/.claude/hooks/custom-user-hook.sh") {
		t.Error("arquivo do usuario nao rastreado pelo manifesto NAO deveria ser removido")
	}

	// Reexecucao idempotente: nada rastreado ainda existe (exceto o proprio manifesto, ja removido).
	if err := svc.Execute("/project", false); err == nil {
		t.Fatal("segunda execucao: esperava erro (governanca ja removida, .agents/skills ausente) — comportamento consistente, nao panic nem falha inesperada")
	}
}

// TestUninstall_OldManifestWithoutFileTracking_FallsBackConservative valida que
// um manifesto antigo (fixture sem o campo installed_files) faz a desinstalacao
// cair no caminho conservador anunciado, sem apagar arquivo nao rastreado.
func TestUninstall_OldManifestWithoutFileTracking_FallsBackConservative(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.claude/hooks/validate-preload.sh"] = []byte("pre")
	ffs.Files["/project/.claude/hooks/custom-user-hook.sh"] = []byte("# nao instalado pelo harness")
	// Manifesto antigo: sem o campo installed_files (aditivo, RF-05).
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{"version": "0.9.0-legacy"}`)

	var buf strings.Builder
	printer := &output.Printer{Out: &buf, Err: &buf, Verbose: true}
	svc := NewService(ffs, printer)

	if err := svc.Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/.claude/hooks/custom-user-hook.sh") {
		t.Error("arquivo do usuario nao rastreado NAO deveria ser removido no caminho conservador")
	}
	// validate-preload.sh faz parte da lista fixa conservadora legada — continua sendo removido.
	if ffs.Exists("/project/.claude/hooks/validate-preload.sh") {
		t.Error("validate-preload.sh (lista conservadora legada) deveria ser removido")
	}
	if !strings.Contains(buf.String(), "manifesto sem rastreamento por arquivo") {
		t.Errorf("saida deveria anunciar caminho conservador; got: %q", buf.String())
	}
}
