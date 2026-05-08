package upgrade

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
)

func setupTestService(ffs *fs.FakeFileSystem) *Service {
	printer := output.New(false)
	mfst := manifest.NewStore(ffs)
	adpt := adapters.NewGenerator(ffs, printer)
	ctxg := contextgen.NewGenerator(ffs, printer)
	return NewService(ffs, printer, mfst, adpt, ctxg)
}

func setupTestServiceWithOutput(ffs *fs.FakeFileSystem, out, errOut *bytes.Buffer) *Service {
	printer := output.New(false)
	printer.Out = out
	printer.Err = errOut
	mfst := manifest.NewStore(ffs)
	adpt := adapters.NewGenerator(ffs, printer)
	ctxg := contextgen.NewGenerator(ffs, printer)
	return NewService(ffs, printer, mfst, adpt, ctxg)
}

func readManifestFromFakeFS(t *testing.T, ffs *fs.FakeFileSystem, path string) manifest.Manifest {
	t.Helper()

	data, err := ffs.ReadFile(path)
	if err != nil {
		t.Fatalf("falha ao ler manifesto %s: %v", path, err)
	}

	var mf manifest.Manifest
	if err := json.Unmarshal(data, &mf); err != nil {
		t.Fatalf("falha ao decodificar manifesto %s: %v", path, err)
	}

	return mf
}

func setVersionForTest(t *testing.T, resolvedVersion string) {
	t.Helper()
	t.Cleanup(version.SetForTest(resolvedVersion))
}

func TestUpgrade_NoSkillsDir(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})

	if err == nil {
		t.Fatal("expected error when .agents/skills/ is missing")
	}
}

func TestUpgrade_CheckOnly_Outdated(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()

	// Source com versao mais nova
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 2.0.0\ndescription: Review.\n---\n")

	// Target com versao antiga
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})

	if err == nil {
		t.Fatal("expected error for outdated skills in check-only mode")
	}
}

func TestUpgrade_CheckOnly_UpToDate(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()

	content := []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpgradeCheck_PrintsInformativeVersionLineWhenManifestDiffers(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "0.11.2",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})
	if err != nil {
		t.Fatalf("upgrade --check falhou: %v", err)
	}

	if !strings.Contains(stdout.String(), "CLI: 0.12.0 (manifesto: 0.11.2)") {
		t.Fatalf("saida nao exibiu linha informativa esperada: %q", stdout.String())
	}
}

func TestUpgradeCheck_OmitsInformativeVersionLineWhenVersionsMatch(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "0.12.0",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})
	if err != nil {
		t.Fatalf("upgrade --check falhou: %v", err)
	}

	if strings.Contains(stdout.String(), "CLI: ") {
		t.Fatalf("saida nao deveria exibir linha informativa: %q", stdout.String())
	}
}

func TestUpgradeCheck_OmitsInformativeVersionLineForNonSemverManifest(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "dev",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})
	if err != nil {
		t.Fatalf("upgrade --check falhou: %v", err)
	}

	if strings.Contains(stdout.String(), "CLI: ") {
		t.Fatalf("saida nao deveria exibir linha informativa para manifesto nao-semver: %q", stdout.String())
	}
}

func TestUpgradeCheck_OmitsInformativeVersionLineForIncompleteSemverManifest(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "1.2",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})
	if err != nil {
		t.Fatalf("upgrade --check falhou: %v", err)
	}

	if strings.Contains(stdout.String(), "CLI: ") {
		t.Fatalf("saida nao deveria exibir linha informativa para manifesto com semver incompleto: %q", stdout.String())
	}
}

func TestUpgradeCheck_OmitsInformativeVersionLineForMalformedPrereleaseManifest(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "1.2.3-",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})
	if err != nil {
		t.Fatalf("upgrade --check falhou: %v", err)
	}

	if strings.Contains(stdout.String(), "CLI: ") {
		t.Fatalf("saida nao deveria exibir linha informativa para prerelease malformado: %q", stdout.String())
	}
}

func TestUpgrade_UsesResolvedExecutableVersionInOutput(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/VERSION"] = []byte("0.11.2")
	content := []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "1.2.3")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	svc := setupTestServiceWithOutput(ffs, &stdout, &stderr)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "ai-spec 1.2.3") {
		t.Fatalf("saida nao exibiu versao resolvida do executavel: %q", got)
	}
	if strings.Contains(got, "ai-spec 0.11.2") {
		t.Fatalf("saida nao deveria exibir VERSION da fonte: %q", got)
	}
}

func TestUpgrade_UpdatesManifestVersionAfterSuccessfulUpgrade(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "0.11.2",
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if mf.Version != "0.12.0" {
		t.Fatalf("version do manifesto nao atualizada: got %q want %q", mf.Version, "0.12.0")
	}
}

func TestUpgrade_RewritesManifestWhenVersionChangesWithoutSkillChanges(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.4.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	initialUpdatedAt := time.Unix(1700000000, 0)
	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "0.11.2",
		CreatedAt: time.Unix(1690000000, 0),
		UpdatedAt: initialUpdatedAt,
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if mf.Version != "0.12.0" {
		t.Fatalf("version do manifesto nao atualizada: got %q want %q", mf.Version, "0.12.0")
	}
	if !mf.UpdatedAt.After(initialUpdatedAt) {
		t.Fatalf("updated_at deveria ter sido regravado: got %s want > %s", mf.UpdatedAt, initialUpdatedAt)
	}
	if got := mf.SkillVersions["review"]; got != "1.4.0" {
		t.Fatalf("skill_versions[review] incorreto: got %q want %q", got, "1.4.0")
	}
}

func TestUpgrade_DoesNotRewriteManifestWhenVersionAndSkillsAreUnchanged(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.4.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:       "0.12.0",
		CreatedAt:     time.Unix(1690000000, 0),
		UpdatedAt:     time.Unix(1700000000, 0),
		Langs:         []skills.Lang{},
		SkillVersions: map[string]string{"review": "1.4.0"},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	before, err := ffs.ReadFile("/project/.ai_spec_harness.json")
	if err != nil {
		t.Fatalf("falha ao ler manifesto antes do upgrade: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	after, err := ffs.ReadFile("/project/.ai_spec_harness.json")
	if err != nil {
		t.Fatalf("falha ao ler manifesto apos upgrade: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("manifesto nao deveria ter sido regravado\nantes: %s\ndepois: %s", string(before), string(after))
	}
}

func TestUpgrade_ReconcilesSkillVersionsAfterSkillUpdate(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.1.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:       "0.11.2",
		CreatedAt:     time.Unix(1690000000, 0),
		UpdatedAt:     time.Unix(1700000000, 0),
		Langs:         []skills.Lang{},
		SkillVersions: map[string]string{"review": "1.0.0"},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if got := mf.SkillVersions["review"]; got != "2.1.0" {
		t.Fatalf("skill_versions[review] nao reconciliado: got %q want %q", got, "2.1.0")
	}
}

func TestUpgrade_PopulatesSkillVersionsForLegacyManifest(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.4.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	legacyManifest := []byte(`{
  "version": "0.11.2",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "langs": []
}`)
	ffs.Files["/project/.ai_spec_harness.json"] = legacyManifest

	setVersionForTest(t, "0.12.0")

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if got := mf.SkillVersions["review"]; got != "1.4.0" {
		t.Fatalf("manifesto legado nao recebeu skill_versions[review]: got %q want %q", got, "1.4.0")
	}
}

func TestUpgrade_RefsChangedFiles(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	printer := output.New(false)
	svc := NewService(ffs, printer, manifest.NewStore(ffs), adapters.NewGenerator(ffs, printer), contextgen.NewGenerator(ffs, printer))

	ffs.Files["/source/references/a.md"] = []byte("new")
	ffs.Files["/source/references/b.md"] = []byte("changed")
	ffs.Files["/project/references/b.md"] = []byte("old")
	ffs.Files["/project/references/c.md"] = []byte("removed")

	changed := svc.refsChangedFiles("/source/references", "/project/references")
	got := strings.Join(changed, "\n")

	for _, want := range []string{"+ a.md (novo)", "~ b.md (modificado)", "- c.md (removido)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("changed refs missing %q in %q", want, got)
		}
	}
}

func TestUpgrade_RegeneratesGovernanceOnSchemaDivergence(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.9.0 -->\n# old")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not regenerated: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema: 1.0.0") {
		t.Fatalf("expected regenerated schema, got %q", string(data))
	}
}

func TestUpgrade_AdaptersRegeneratedAfterSkillChange(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Dirs["/project/.claude"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/.claude/agents/reviewer.md") {
		t.Error("expected reviewer.md to be regenerated after skill update")
	}
}

func TestUpgrade_CodexRegeneratedAfterSkillChange(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.codex/config.toml"] = []byte("stale content")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("codex config should still exist: %v", err)
	}
	if strings.Contains(string(data), "stale content") {
		t.Error("codex config should have been regenerated, still contains stale content")
	}
	if !strings.Contains(string(data), ".agents/skills/agent-governance") {
		t.Errorf("codex config missing expected skills, got: %q", string(data))
	}
}

func TestUpgrade_PreservesCustomClaudeSettings(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Dirs["/project/.claude"] = true

	customSettings := `{"hooks": {"PreToolUse": []}, "customOption": "value"}`
	ffs.Files["/project/.claude/settings.local.json"] = []byte(customSettings)

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("settings.local.json should be preserved: %v", err)
	}
	if string(data) != customSettings {
		t.Fatalf("settings.local.json was modified by upgrade, got: %q", string(data))
	}
}

func TestUpgrade_CrossVersionSchemaChange(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.1.0 -->\n# muito antigo")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not found after upgrade: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema: 1.0.0") {
		t.Fatalf("expected governance-schema: 1.0.0 after upgrade, got: %q", string(data))
	}
	if strings.Contains(string(data), "muito antigo") {
		t.Fatal("old AGENTS.md content should have been replaced by regeneration")
	}
}

func TestUpgrade_PreservesAgentsLocal(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.9.0 -->\n# content")

	const localContent = "# Custom user section\n\nPersonal rules here.\n"
	ffs.Files["/project/AGENTS.local.md"] = []byte(localContent)

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.local.md")
	if err != nil {
		t.Fatalf("AGENTS.local.md should be preserved: %v", err)
	}
	if string(data) != localContent {
		t.Fatalf("AGENTS.local.md was modified: got %q", string(data))
	}

	agentsData, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not found after upgrade: %v", err)
	}
	if !strings.Contains(string(agentsData), "governance-schema: 1.0.0") {
		t.Fatalf("AGENTS.md should have been regenerated, got: %q", string(agentsData))
	}
}

func TestUpgrade_RecopiesT02Artifacts(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.claude/scripts/validate-bugfix-evidence.sh"] = []byte("#!/usr/bin/env bash\n# new bugfix")
	ffs.Files["/source/.claude/scripts/validate-refactor-evidence.sh"] = []byte("#!/usr/bin/env bash\n# new refactor")
	ffs.Files["/source/scripts/lib/check-invocation-depth.sh"] = []byte("#!/usr/bin/env bash\n# new depth")
	ffs.Files["/project/.claude/scripts/validate-bugfix-evidence.sh"] = []byte("#!/usr/bin/env bash\n# old bugfix")
	ffs.Files["/project/.claude/scripts/validate-refactor-evidence.sh"] = []byte("#!/usr/bin/env bash\n# old refactor")
	ffs.Files["/project/scripts/lib/check-invocation-depth.sh"] = []byte("#!/usr/bin/env bash\n# old depth")
	ffs.Dirs["/project/.claude"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bugfix, err := ffs.ReadFile("/project/.claude/scripts/validate-bugfix-evidence.sh")
	if err != nil || !strings.Contains(string(bugfix), "# new bugfix") {
		t.Fatalf("expected validate-bugfix-evidence.sh synced, got %q err=%v", string(bugfix), err)
	}

	refactor, err := ffs.ReadFile("/project/.claude/scripts/validate-refactor-evidence.sh")
	if err != nil || !strings.Contains(string(refactor), "# new refactor") {
		t.Fatalf("expected validate-refactor-evidence.sh synced, got %q err=%v", string(refactor), err)
	}

	depth, err := ffs.ReadFile("/project/scripts/lib/check-invocation-depth.sh")
	if err != nil || !strings.Contains(string(depth), "# new depth") {
		t.Fatalf("expected check-invocation-depth.sh synced, got %q err=%v", string(depth), err)
	}
}

func TestUpgrade_RecopiesGeminiHook(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.gemini/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash\n# new gemini hook")
	ffs.Files["/project/.gemini/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash\n# old gemini hook")
	ffs.Dirs["/project/.gemini"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.gemini/hooks/validate-preload.sh")
	if err != nil {
		t.Fatalf("gemini hook should exist after upgrade: %v", err)
	}
	if !strings.Contains(string(data), "# new gemini hook") {
		t.Fatalf("expected gemini hook to be re-copied, got %q", string(data))
	}
}

func TestUpgrade_RegeneratesAdaptersAndSupportFiles(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.claude/rules/governance.md"] = []byte("# new governance")
	ffs.Files["/source/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash\n# new")
	ffs.Files["/project/.claude/rules/governance.md"] = []byte("# old governance")
	ffs.Files["/project/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash\n# old")
	ffs.Files["/project/.codex/config.toml"] = []byte("stale")
	ffs.Dirs["/project/.claude"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rules, err := ffs.ReadFile("/project/.claude/rules/governance.md")
	if err != nil || string(rules) != "# new governance" {
		t.Fatalf("expected governance.md synced, got %q err=%v", string(rules), err)
	}

	script, err := ffs.ReadFile("/project/.claude/scripts/validate-task-evidence.sh")
	if err != nil || !strings.Contains(string(script), "# new") {
		t.Fatalf("expected validate-task-evidence synced, got %q err=%v", string(script), err)
	}

	codex, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("expected codex config regenerated: %v", err)
	}
	if !strings.Contains(string(codex), ".agents/skills/agent-governance") {
		t.Fatalf("expected codex config regenerated, got %q", string(codex))
	}
}

func setupOSTestService() *Service {
	printer := output.New(false)
	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	return NewService(fsys, printer, mfst, adpt, ctxg)
}

func TestUpgrade_EmbeddedSource_NoSourceFlag(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()

	// Simular projeto com skills em versao antiga instaladas
	skillsDir := projectDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldContent := []byte("---\nversion: 0.0.1\ndescription: Review.\n---\n# Review\n")
	if err := os.WriteFile(skillsDir+"/SKILL.md", oldContent, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := setupOSTestService()

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  "", // sem --source: usa embutido
		CheckOnly:  true,
	})

	// Com skill em versao antiga e embutido mais novo, deve detectar desatualizacao
	// (ou ao menos nao falhar com "fonte nao encontrada")
	if err != nil && strings.Contains(err.Error(), "nao encontrado") {
		t.Fatalf("upgrade sem --source falhou por falta de fonte: %v", err)
	}
	// O erro esperado e de skills desatualizadas, nao de configuracao
	if err != nil && !strings.Contains(err.Error(), "desatualizad") {
		t.Logf("upgrade retornou: %v", err)
	}
}

func TestUpgrade_EmbeddedSource_UpdatesSkills(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()

	// Criar estrutura de projeto com skills em versao muito antiga
	skillsDir := projectDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldContent := []byte("---\nversion: 0.0.1\ndescription: Old Review.\n---\n# Old Review\n")
	if err := os.WriteFile(skillsDir+"/SKILL.md", oldContent, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := setupOSTestService()

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  "", // usa embutido
		CheckOnly:  false,
	})

	// Upgrade nao deve falhar com erro de configuracao
	if err != nil && strings.Contains(err.Error(), "nao encontrado") {
		t.Fatalf("upgrade sem --source falhou por falta de fonte: %v", err)
	}

	// A skill deve ter sido atualizada para a versao embutida
	data, err := os.ReadFile(skillsDir + "/SKILL.md")
	if err != nil {
		t.Fatalf("SKILL.md nao encontrado apos upgrade: %v", err)
	}
	if strings.Contains(string(data), "0.0.1") {
		t.Error("skill nao foi atualizada pela versao embutida")
	}
}


func TestCheckSchemaDivergence_FallsBackToConstantWhenTemplateHoldsPlaceholder(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	svc := setupTestServiceWithOutput(ffs, out, errOut)

	projectDir := "/proj"
	sourceDir := "/src"

	if err := ffs.MkdirAll(projectDir); err != nil {
		t.Fatal(err)
	}
	projectAgents := "<!-- governance-schema: " + contextgen.GovernanceSchemaVersion + " -->\n# AGENTS\n"
	if err := ffs.WriteFile(projectDir+"/AGENTS.md", []byte(projectAgents)); err != nil {
		t.Fatal(err)
	}

	templatePath := sourceDir + "/.agents/skills/analyze-project/assets/agents-template.md"
	if err := ffs.MkdirAll(sourceDir + "/.agents/skills/analyze-project/assets"); err != nil {
		t.Fatal(err)
	}
	if err := ffs.WriteFile(templatePath, []byte("<!-- governance-schema: {{GOVERNANCE_SCHEMA_VERSION}} -->\n")); err != nil {
		t.Fatal(err)
	}

	if got := svc.checkSchemaDivergence(sourceDir, projectDir); got {
		t.Fatalf("checkSchemaDivergence retornou divergente; esperava nao-divergente quando template tem placeholder e projeto usa versao corrente. stdout=%q", out.String())
	}
}

func TestCheckSchemaDivergence_DetectsRealDivergenceAgainstConstant(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	svc := setupTestServiceWithOutput(ffs, out, errOut)

	projectDir := "/proj"
	sourceDir := "/src"

	if err := ffs.MkdirAll(projectDir); err != nil {
		t.Fatal(err)
	}
	if err := ffs.WriteFile(projectDir+"/AGENTS.md", []byte("<!-- governance-schema: 0.0.1-stale -->\n")); err != nil {
		t.Fatal(err)
	}
	if err := ffs.MkdirAll(sourceDir + "/.agents/skills/analyze-project/assets"); err != nil {
		t.Fatal(err)
	}
	if err := ffs.WriteFile(sourceDir+"/.agents/skills/analyze-project/assets/agents-template.md",
		[]byte("<!-- governance-schema: {{GOVERNANCE_SCHEMA_VERSION}} -->\n")); err != nil {
		t.Fatal(err)
	}

	if got := svc.checkSchemaDivergence(sourceDir, projectDir); !got {
		t.Fatalf("checkSchemaDivergence retornou nao-divergente; esperava divergencia real. stdout=%q", out.String())
	}
}
