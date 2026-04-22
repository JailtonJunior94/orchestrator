package aispecharness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionNoFlag verifica RF-10: sem flag, a saida exibe apenas a versao do CLI.
func TestVersionNoFlag(t *testing.T) {
	var sb strings.Builder
	if err := runVersion(&sb, "", ""); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.HasPrefix(out, "ai-spec-harness ") {
		t.Errorf("saida esperada com prefixo 'ai-spec-harness ', obteve: %q", out)
	}
	if strings.Contains(out, "Embedded skills:") || strings.Contains(out, "Installed skills:") {
		t.Errorf("sem flag nao deve exibir secoes de skills, obteve: %q", out)
	}
}

// TestVersionSkillsEmbedded verifica RF-11a: --skills=embedded lista skills embutidas.
func TestVersionSkillsEmbedded(t *testing.T) {
	var sb strings.Builder
	if err := runVersion(&sb, "embedded", ""); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Embedded skills:") {
		t.Errorf("esperado cabecalho 'Embedded skills:', obteve: %q", out)
	}
	if strings.Contains(out, "Installed skills:") {
		t.Errorf("modo embedded nao deve exibir secao 'Installed skills:', obteve: %q", out)
	}
	// Deve haver ao menos uma skill listada (execute-task existe no embedded FS)
	if !strings.Contains(out, "execute-task") {
		t.Errorf("esperado 'execute-task' na lista de skills embutidas, obteve: %q", out)
	}
}

// TestVersionSkillsInstalled verifica RF-11b: --skills=installed lista skills do projeto.
func TestVersionSkillsInstalled(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".agents", "skills", "minha-skill")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("criar diretorio de skill: %v", err)
	}
	skillMd := `---
name: minha-skill
version: 2.0.0
description: Skill de teste
---

# Minha Skill
`
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
		t.Fatalf("escrever SKILL.md: %v", err)
	}

	var sb strings.Builder
	if err := runVersion(&sb, "installed", dir); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Installed skills:") {
		t.Errorf("esperado cabecalho 'Installed skills:', obteve: %q", out)
	}
	if strings.Contains(out, "Embedded skills:") {
		t.Errorf("modo installed nao deve exibir secao 'Embedded skills:', obteve: %q", out)
	}
	if !strings.Contains(out, "minha-skill") {
		t.Errorf("esperado 'minha-skill' na lista de skills instaladas, obteve: %q", out)
	}
	if !strings.Contains(out, "2.0.0") {
		t.Errorf("esperado versao '2.0.0' na lista de skills instaladas, obteve: %q", out)
	}
}

// TestVersionSkillsBoth verifica RF-11c: --skills (sem valor, NoOptDefVal="both") lista ambas as secoes.
func TestVersionSkillsBoth(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".agents", "skills", "outra-skill")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("criar diretorio de skill: %v", err)
	}
	skillMd := `---
name: outra-skill
version: 1.0.0
description: Outra skill de teste
---

# Outra Skill
`
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
		t.Fatalf("escrever SKILL.md: %v", err)
	}

	var sb strings.Builder
	if err := runVersion(&sb, "both", dir); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Embedded skills:") {
		t.Errorf("esperado cabecalho 'Embedded skills:', obteve: %q", out)
	}
	if !strings.Contains(out, "Installed skills:") {
		t.Errorf("esperado cabecalho 'Installed skills:', obteve: %q", out)
	}
}

// TestVersionSkillsNoVersion verifica RF-12: skill sem campo version exibe "(sem versao)".
func TestVersionSkillsNoVersion(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".agents", "skills", "skill-sem-versao")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("criar diretorio de skill: %v", err)
	}
	// Frontmatter sem campo version
	skillMd := `---
name: skill-sem-versao
description: Skill sem campo version no frontmatter
---

# Skill Sem Versao
`
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
		t.Fatalf("escrever SKILL.md: %v", err)
	}

	var sb strings.Builder
	if err := runVersion(&sb, "installed", dir); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "(sem versao)") {
		t.Errorf("esperado '(sem versao)' para skill sem campo version, obteve: %q", out)
	}
}

// TestVersionSkillsInstalledEmpty verifica que diretorio sem skills exibe mensagem adequada.
func TestVersionSkillsInstalledEmpty(t *testing.T) {
	dir := t.TempDir()

	var sb strings.Builder
	if err := runVersion(&sb, "installed", dir); err != nil {
		t.Fatalf("runVersion erro inesperado: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Installed skills:") {
		t.Errorf("esperado cabecalho 'Installed skills:', obteve: %q", out)
	}
	if !strings.Contains(out, "nenhuma skill instalada") {
		t.Errorf("esperado 'nenhuma skill instalada' para diretorio vazio, obteve: %q", out)
	}
}

// TestVersionSkillsEmbeddedSorted verifica que skills embutidas sao exibidas em ordem alfabetica.
func TestVersionSkillsEmbeddedSorted(t *testing.T) {
	skills, err := listEmbeddedSkills()
	if err != nil {
		t.Fatalf("listEmbeddedSkills erro: %v", err)
	}
	if len(skills) < 2 {
		t.Skip("menos de 2 skills embutidas — ordenacao nao verificavel")
	}
	for i := 1; i < len(skills); i++ {
		if skills[i].Name < skills[i-1].Name {
			t.Errorf("skills embutidas fora de ordem: %q vem antes de %q", skills[i-1].Name, skills[i].Name)
		}
	}
}
