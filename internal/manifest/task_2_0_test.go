package manifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestManifest_SkillVersions_Retrocompatibility(t *testing.T) {
	// Teste 1: Deserializar JSON sem "skill_versions" — campo nil, sem erro
	jsonData := `{
		"version": "dev",
		"created_at": "2026-04-22T12:00:00Z",
		"updated_at": "2026-04-22T12:00:00Z",
		"source_dir": "/tmp",
		"link_mode": "symlink",
		"tools": [],
		"langs": [],
		"skills": ["skill1"],
		"checksums": {"skill1": "abc"}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		t.Fatalf("falha ao deserializar manifesto antigo: %v", err)
	}

	if m.SkillVersions != nil {
		t.Errorf("esperava SkillVersions nil para manifesto antigo, obteve: %v", m.SkillVersions)
	}
}

func TestManifest_SkillVersions_Serialization(t *testing.T) {
	// Teste 2: Serializar Manifest com SkillVersions == nil — campo ausente do JSON (omitempty)
	m1 := Manifest{
		Version: "1.0.0",
	}
	data1, err := json.Marshal(m1)
	if err != nil {
		t.Fatalf("falha ao serializar m1: %v", err)
	}
	if strings.Contains(string(data1), "skill_versions") {
		t.Errorf("esperava que skill_versions fosse omitido, mas foi encontrado: %s", string(data1))
	}

	// Teste 3: Serializar Manifest com SkillVersions preenchido — campo presente no JSON com valores corretos
	m2 := Manifest{
		Version: "1.0.0",
		SkillVersions: map[string]string{
			"skill1": "1.0.0",
			"skill2": "1.1.0",
		},
	}
	data2, err := json.Marshal(m2)
	if err != nil {
		t.Fatalf("falha ao serializar m2: %v", err)
	}
	if !strings.Contains(string(data2), `"skill_versions":{"skill1":"1.0.0","skill2":"1.1.0"}`) {
		t.Errorf("esperava skill_versions no JSON, obteve: %s", string(data2))
	}
}
