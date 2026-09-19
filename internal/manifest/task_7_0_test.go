package manifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestManifest_FileChecksums_Retrocompatibility(t *testing.T) {
	jsonData := `{
		"version": "2.0.1",
		"created_at": "2026-04-22T12:00:00Z",
		"updated_at": "2026-04-22T12:00:00Z",
		"source_dir": "/tmp",
		"link_mode": "symlink",
		"tools": [],
		"langs": [],
		"skills": ["skill1"],
		"checksums": {"skill1": "abc"},
		"installed_files": ["AGENTS.md", "CLAUDE.md"],
		"merged_files": []
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		t.Fatalf("falha ao deserializar manifesto da v2.0.1: %v", err)
	}

	if m.FileChecksums != nil {
		t.Errorf("esperava FileChecksums nil para manifesto legado, obteve: %v", m.FileChecksums)
	}
	if !m.HasFileTracking() {
		t.Error("manifesto legado com installed_files deveria continuar reportando HasFileTracking() == true")
	}
}

func TestManifest_FileChecksums_LegacyManifestWithoutAnyTrackingNeverDiverges(t *testing.T) {
	jsonData := `{
		"version": "1.5.0",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"source_dir": "/tmp",
		"link_mode": "copy",
		"tools": [],
		"langs": [],
		"skills": ["skill1"],
		"checksums": {"skill1": "abc"}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		t.Fatalf("falha ao deserializar manifesto pre-RF-05: %v", err)
	}

	if m.HasFileTracking() {
		t.Error("manifesto sem installed_files nao pode reportar rastreio de arquivos")
	}
	if m.FileChecksums != nil {
		t.Errorf("esperava FileChecksums nil, obteve: %v", m.FileChecksums)
	}
	if _, ok := m.FileChecksums["qualquer.md"]; ok {
		t.Error("leitura de checksum em manifesto legado nunca pode ser encontrada")
	}
}

func TestManifest_FileChecksums_Serialization(t *testing.T) {
	m1 := Manifest{Version: "1.0.0"}
	data1, err := json.Marshal(m1)
	if err != nil {
		t.Fatalf("falha ao serializar m1: %v", err)
	}
	if strings.Contains(string(data1), "file_checksums") {
		t.Errorf("esperava que file_checksums fosse omitido, mas foi encontrado: %s", string(data1))
	}

	m2 := Manifest{
		Version: "1.0.0",
		FileChecksums: map[string]string{
			"AGENTS.md": "deadbeef",
		},
	}
	data2, err := json.Marshal(m2)
	if err != nil {
		t.Fatalf("falha ao serializar m2: %v", err)
	}
	if !strings.Contains(string(data2), `"file_checksums":{"AGENTS.md":"deadbeef"}`) {
		t.Errorf("esperava file_checksums no JSON, obteve: %s", string(data2))
	}
}
