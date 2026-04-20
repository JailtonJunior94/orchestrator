package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLogEntries(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	old := now.Add(-48 * time.Hour)

	tests := []struct {
		name        string
		fileContent string // vazio = arquivo não criado
		since       time.Duration
		wantLen     int
		wantEntries []logEntry
	}{
		{
			name:    "arquivo_ausente",
			since:   0,
			wantLen: 0,
		},
		{
			name:        "linha_valida_com_ref",
			fileContent: now.Format(time.RFC3339) + " skill=bugfix ref=testing.md\n",
			since:       0,
			wantLen:     1,
			wantEntries: []logEntry{
				{Timestamp: now, Skill: "bugfix", Ref: "testing.md"},
			},
		},
		{
			name:        "linha_valida_sem_ref",
			fileContent: now.Format(time.RFC3339) + " skill=review\n",
			since:       0,
			wantLen:     1,
			wantEntries: []logEntry{
				{Timestamp: now, Skill: "review", Ref: ""},
			},
		},
		{
			name:        "linha_malformada_ignorada",
			fileContent: "invalido\n" + now.Format(time.RFC3339) + " skill=bugfix\n",
			since:       0,
			wantLen:     1,
		},
		{
			name:        "filtro_since_exclui_antiga",
			fileContent: old.Format(time.RFC3339) + " skill=bugfix ref=old.md\n",
			since:       1 * time.Hour,
			wantLen:     0,
		},
		{
			name:        "filtro_since_inclui_recente",
			fileContent: now.Format(time.RFC3339) + " skill=review ref=recent.md\n",
			since:       1 * time.Hour,
			wantLen:     1,
		},
		{
			name:        "multiplas_entradas",
			fileContent: strings.Repeat(now.Format(time.RFC3339)+" skill=foo ref=bar.md\n", 5),
			since:       0,
			wantLen:     5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "telemetry.log")

			if tc.fileContent != "" {
				if err := os.WriteFile(logPath, []byte(tc.fileContent), 0644); err != nil {
					t.Fatalf("escrever arquivo de log: %v", err)
				}
			}

			entries, err := parseLogEntries(logPath, tc.since)
			if err != nil {
				t.Fatalf("parseLogEntries retornou erro inesperado: %v", err)
			}
			if len(entries) != tc.wantLen {
				t.Errorf("esperava %d entradas, obteve %d", tc.wantLen, len(entries))
			}
			for i, want := range tc.wantEntries {
				if i >= len(entries) {
					t.Fatalf("entrada %d ausente nos resultados", i)
				}
				got := entries[i]
				if !got.Timestamp.Equal(want.Timestamp) {
					t.Errorf("entrada[%d].Timestamp: queria %v, obteve %v", i, want.Timestamp, got.Timestamp)
				}
				if got.Skill != want.Skill {
					t.Errorf("entrada[%d].Skill: queria %q, obteve %q", i, want.Skill, got.Skill)
				}
				if got.Ref != want.Ref {
					t.Errorf("entrada[%d].Ref: queria %q, obteve %q", i, want.Ref, got.Ref)
				}
			}
		})
	}
}
