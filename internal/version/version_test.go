package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadVersionFile_Exists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("  1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := ReadVersionFile(dir)
	if got != "1.2.3" {
		t.Errorf("ReadVersionFile: got %q, want %q", got, "1.2.3")
	}
}

func TestReadVersionFile_Missing(t *testing.T) {
	dir := t.TempDir()
	got := ReadVersionFile(dir)
	if got != "unknown" {
		t.Errorf("ReadVersionFile: got %q, want %q", got, "unknown")
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name        string
		ldflags     string
		fileContent string // vazio = nao criar arquivo
		want        string
	}{
		{
			name:    "ldflags definido: retorna versao injetada",
			ldflags: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:        "dev com VERSION file: retorna versao-dev",
			ldflags:     "dev",
			fileContent: "0.9.2",
			want:        "0.9.2-dev",
		},
		{
			name:    "dev sem VERSION file: retorna dev",
			ldflags: "dev",
			want:    "dev",
		},
	}

	orig := Version
	t.Cleanup(func() { Version = orig })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			Version = tc.ldflags
			dir := t.TempDir()
			if tc.fileContent != "" {
				if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte(tc.fileContent+"\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			got := Resolve(dir)
			if got != tc.want {
				t.Errorf("Resolve: got %q, want %q", got, tc.want)
			}
		})
	}
}
