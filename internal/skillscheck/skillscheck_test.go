package skillscheck_test

import (
	"encoding/json"
	"io"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
)

func newService(t *testing.T) (*skillscheck.Service, *fs.FakeFileSystem) {
	t.Helper()
	fake := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	svc := skillscheck.NewService(fake, printer)
	return svc, fake
}

func writeLock(t *testing.T, fake *fs.FakeFileSystem, projectDir string, entries map[string]skillscheck.LockEntry) {
	t.Helper()
	lock := skillscheck.LockFile{Version: 1, Skills: entries}
	data, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile(filepath.Join(projectDir, "skills-lock.json"), data); err != nil {
		t.Fatalf("WriteFile skills-lock.json: %v", err)
	}
}

func writeSkillMD(t *testing.T, fake *fs.FakeFileSystem, projectDir, skillName, version string) {
	t.Helper()
	content := []byte("---\nname: " + skillName + "\nversion: " + version + "\n---\n# Skill\n")
	path := filepath.Join(projectDir, ".agents", "skills", skillName, "SKILL.md")
	if err := fake.WriteFile(path, content); err != nil {
		t.Fatalf("WriteFile SKILL.md: %v", err)
	}
}

func TestCheck_CompatibleUpgrade(t *testing.T) {
	svc, fake := newService(t)
	dir := "/proj"

	writeLock(t, fake, dir, map[string]skillscheck.LockEntry{
		"my-skill": {Version: "1.0.0"},
	})
	writeSkillMD(t, fake, dir, "my-skill", "1.2.0")

	results, err := svc.Check(dir)
	if err != nil {
		t.Fatalf("Check erro inesperado: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("esperado 1 resultado, obtido %d", len(results))
	}
	if results[0].Drift != skillscheck.DriftMinor {
		t.Errorf("drift esperado=%s, obtido=%s", skillscheck.DriftMinor, results[0].Drift)
	}
	if results[0].Breaking {
		t.Error("upgrade minor nao deve ser breaking")
	}
}

func TestCheck_BreakingUpgrade(t *testing.T) {
	svc, fake := newService(t)
	dir := "/proj"

	writeLock(t, fake, dir, map[string]skillscheck.LockEntry{
		"my-skill": {Version: "1.0.0"},
	})
	writeSkillMD(t, fake, dir, "my-skill", "2.0.0")

	results, err := svc.Check(dir)
	if err != nil {
		t.Fatalf("Check erro inesperado: %v", err)
	}
	if results[0].Drift != skillscheck.DriftBreaking {
		t.Errorf("drift esperado=%s, obtido=%s", skillscheck.DriftBreaking, results[0].Drift)
	}
	if !results[0].Breaking {
		t.Error("major bump deve ser breaking")
	}
}

func TestCheck_VersionIgual(t *testing.T) {
	svc, fake := newService(t)
	dir := "/proj"

	writeLock(t, fake, dir, map[string]skillscheck.LockEntry{
		"my-skill": {Version: "1.0.0"},
	})
	writeSkillMD(t, fake, dir, "my-skill", "1.0.0")

	results, err := svc.Check(dir)
	if err != nil {
		t.Fatalf("Check erro inesperado: %v", err)
	}
	if results[0].Drift != skillscheck.DriftNone {
		t.Errorf("drift esperado=%s, obtido=%s", skillscheck.DriftNone, results[0].Drift)
	}
}

func TestCheck_SkillNaoInstalada(t *testing.T) {
	svc, fake := newService(t)
	dir := "/proj"

	writeLock(t, fake, dir, map[string]skillscheck.LockEntry{
		"missing-skill": {Version: "1.0.0"},
	})
	// nao escreve SKILL.md

	results, err := svc.Check(dir)
	if err != nil {
		t.Fatalf("Check erro inesperado: %v", err)
	}
	if results[0].Drift != skillscheck.DriftNoSkill {
		t.Errorf("drift esperado=%s, obtido=%s", skillscheck.DriftNoSkill, results[0].Drift)
	}
}

func TestCheck_VersaoDesconhecida(t *testing.T) {
	svc, fake := newService(t)
	dir := "/proj"

	writeLock(t, fake, dir, map[string]skillscheck.LockEntry{
		"my-skill": {Version: ""},
	})
	writeSkillMD(t, fake, dir, "my-skill", "1.0.0")

	results, err := svc.Check(dir)
	if err != nil {
		t.Fatalf("Check erro inesperado: %v", err)
	}
	if results[0].Drift != skillscheck.DriftUnknown {
		t.Errorf("drift esperado=%s, obtido=%s", skillscheck.DriftUnknown, results[0].Drift)
	}
}
