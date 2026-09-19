package doctor

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestBuildProviderBlocks_OnlyInstalledProvidersAppear(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "go-implementation", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
		{Tool: "", Skill: ".agents/scripts/validate-task-evidence.sh", State: install.VerifyStateCurrent, Kind: install.VerifyKindArtifact},
	}

	blocks := svc.buildProviderBlocks("/project", items, false)
	if len(blocks) != 1 {
		t.Fatalf("esperava 1 bloco de provedor (apenas claude tem itens), obteve %d: %+v", len(blocks), blocks)
	}
	if blocks[0].Tool != skills.ToolClaude {
		t.Errorf("Tool = %q, want claude", blocks[0].Tool)
	}
	if blocks[0].Name != "Claude (ACP)" {
		t.Errorf("Name = %q, want %q", blocks[0].Name, "Claude (ACP)")
	}
	if len(blocks[0].Checks) != 5 {
		t.Errorf("esperava 5 checks por bloco de provedor (RF-32), obteve %d", len(blocks[0].Checks))
	}
}

func TestBuildProviderBlocks_StableCanonicalOrder(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolOpenCode, Skill: "x", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
		{Tool: skills.ToolClaude, Skill: "x", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
		{Tool: skills.ToolCodex, Skill: "x", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
	}

	blocks := svc.buildProviderBlocks("/project", items, false)
	if len(blocks) != 3 {
		t.Fatalf("esperava 3 blocos, obteve %d", len(blocks))
	}
	wantOrder := []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolOpenCode}
	for i, want := range wantOrder {
		if blocks[i].Tool != want {
			t.Errorf("blocks[%d].Tool = %q, want %q (ordem canonica de skills.AllTools)", i, blocks[i].Tool, want)
		}
	}
}

func TestBuildProviderBlocks_AllFourProvidersWhenInstalled(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	var items []install.VerifyItem
	for _, tool := range skills.AllTools {
		items = append(items, install.VerifyItem{
			Tool: tool, Skill: "go-implementation", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill,
		})
	}

	blocks := svc.buildProviderBlocks("/project", items, false)
	if len(blocks) != len(skills.AllTools) {
		t.Fatalf("esperava %d blocos (um por provedor instalado), obteve %d", len(skills.AllTools), len(blocks))
	}
	for i, tool := range skills.AllTools {
		if blocks[i].Tool != tool {
			t.Errorf("blocks[%d].Tool = %q, want %q", i, blocks[i].Tool, tool)
		}
	}
}

func TestCheckProviderSkills_MissingFails(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "a", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
		{Tool: skills.ToolClaude, Skill: "b", State: install.VerifyStateMissing, Kind: install.VerifyKindSkill},
	}
	check := svc.checkProviderSkills(items)
	if check.Status != "fail" {
		t.Errorf("Status = %q, want fail quando ha skill ausente", check.Status)
	}
	if check.Layer != LayerInstallation {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerInstallation)
	}
}

func TestCheckProviderSkills_DriftedWarns(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "a", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
		{Tool: skills.ToolClaude, Skill: "b", State: install.VerifyStateDrifted, Kind: install.VerifyKindSkill},
	}
	check := svc.checkProviderSkills(items)
	if check.Status != "warn" {
		t.Errorf("Status = %q, want warn quando ha skill divergente", check.Status)
	}
	if check.Layer != LayerAdapter {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerAdapter)
	}
}

func TestCheckProviderSkills_AllCurrentOK(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "a", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
	}
	check := svc.checkProviderSkills(items)
	if check.Status != "ok" {
		t.Errorf("Status = %q, want ok", check.Status)
	}
}

func TestCheckProviderPolicies_MissingFails(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: ".claude/hooks/validate-preload.sh", State: install.VerifyStateMissing, Kind: install.VerifyKindArtifact},
	}
	check := svc.checkProviderPolicies(items)
	if check.Status != "fail" {
		t.Errorf("Status = %q, want fail", check.Status)
	}
	if check.Layer != LayerInstallation {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerInstallation)
	}
}

func TestCheckProviderPreconditions_UnmetFails(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolCodex, Skill: "trusted-hash", State: install.VerifyStateInert, Kind: install.VerifyKindPrecondition},
	}
	check := svc.checkProviderPreconditions(items)
	if check.Status != "fail" {
		t.Errorf("Status = %q, want fail quando precondicao esta inert", check.Status)
	}
	if check.Layer != LayerProvider {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerProvider)
	}
}

func TestCheckProviderPreconditions_UnknownNeverFails(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolCodex, Skill: "trusted-hash", State: install.VerifyStateUnknown, Kind: install.VerifyKindPrecondition},
	}
	check := svc.checkProviderPreconditions(items)
	if check.Status == "fail" {
		t.Errorf("Status = %q, precondicao unknown (ausencia de informacao) nunca deve falhar", check.Status)
	}
}

func TestCheckProviderPreconditions_BinaryMissingWarns(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "claude-agent-acp", State: install.VerifyStateMissing, Kind: install.VerifyKindBinary},
	}
	check := svc.checkProviderPreconditions(items)
	if check.Status != "warn" {
		t.Errorf("Status = %q, want warn quando binario ACP ausente", check.Status)
	}
}

func TestCheckCanonicalSync_DriftedFailsWithAdapterLayer(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	items := []install.VerifyItem{
		{Tool: skills.ToolClaude, Skill: "go-implementation", State: install.VerifyStateDrifted, Kind: install.VerifyKindSkill},
	}
	check := svc.checkCanonicalSync(items, nil)
	if check.Status != "fail" {
		t.Errorf("Status = %q, want fail", check.Status)
	}
	if check.Layer != LayerAdapter {
		t.Errorf("Layer = %q, want %q (a divergencia vive no artefato derivado, nao no core)", check.Layer, LayerAdapter)
	}
	if !strings.Contains(check.Detail, "go-implementation") {
		t.Errorf("Detail deveria nomear o artefato divergente; obteve %q", check.Detail)
	}
}

func TestCheckCanonicalSync_NoItemsIsOK(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	check := svc.checkCanonicalSync(nil, nil)
	if check.Status != "ok" {
		t.Errorf("Status = %q, want ok quando nao ha itens divergentes", check.Status)
	}
	if check.Layer != LayerCore {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerCore)
	}
}

func TestCheckCanonicalSync_VerifyErrorWarns(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	check := svc.checkCanonicalSync(nil, errBoom)
	if check.Status != "warn" {
		t.Errorf("Status = %q, want warn quando o proprio Verify falhou (degradacao honesta, nunca fail silencioso)", check.Status)
	}
}

func TestCheckSkillIntegrity_ConsumesRF60LockMismatch(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project/.agents/skills/bugfix"] = true
	fake.Files["/project/.agents/skills/bugfix/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\nconteudo")
	fake.Files["/project/skills-lock.json"] = []byte(`{
		"version": 1,
		"skills": {
			"bugfix": {"source": "x", "sourceType": "git", "version": "1.0.0", "computedHash": "0000000000000000000000000000000000000000000000000000000000000000"}
		}
	}`)

	svc := setupService(fake, true)
	check := svc.checkSkillIntegrity("/project")
	if check.Status != "fail" {
		t.Errorf("Status = %q, want fail: hash do lock diverge do SKILL.md instalado (consumo de RF-60, subtarefa 11.8)", check.Status)
	}
	if check.Layer != LayerValidation {
		t.Errorf("Layer = %q, want %q", check.Layer, LayerValidation)
	}
}

func TestCheckSkillIntegrity_NoLockFileWarns(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)
	check := svc.checkSkillIntegrity("/project")
	if check.Status != "warn" {
		t.Errorf("Status = %q, want warn quando skills-lock.json nao existe (nao e um invariante quebrado, e ausencia de mecanismo)", check.Status)
	}
}

func TestPrintChecks_CountsOnlyFailStatus(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)

	checks := []Check{
		{Name: "a", Status: "ok", Layer: LayerCore},
		{Name: "b", Status: "warn", Layer: LayerCore},
		{Name: "c", Status: "fail", Layer: LayerCore},
		{Name: "d", Status: "fail", Layer: LayerCore},
	}
	got := svc.printChecks(checks)
	if got != 2 {
		t.Errorf("printChecks = %d, want 2 (apenas Status==fail conta)", got)
	}
}

var errBoom = fakeErr("boom")

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
