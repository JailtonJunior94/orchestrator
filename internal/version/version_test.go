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

func TestResolveFromExecutable(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })

	t.Run("ldflags setado: retorna ldflags sem tocar filesystem", func(t *testing.T) {
		Version = "1.1.0-test"
		got := ResolveFromExecutable()
		if got != "1.1.0-test" {
			t.Errorf("ResolveFromExecutable: got %q, want %q", got, "1.1.0-test")
		}
	})

	t.Run("dev sem VERSION file: retorna dev", func(t *testing.T) {
		Version = "dev"
		// Em ambiente de teste, dificilmente havera um arquivo VERSION adjacente ao binario do teste
		// a menos que a gente o crie.
		
		// Para garantir o Teste 2, vamos tentar criar o VERSION adjacente ao executavel.
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		resolved, _ := filepath.EvalSymlinks(exe)
		dir := filepath.Dir(resolved)
		
		versionFile := filepath.Join(dir, "VERSION")
		
		// Se o arquivo ja existir, vamos tentar nao sobrescrever se for importante, 
		// mas em dir de build/temp deve ser ok.
		// Na verdade, em `go test`, o dir e temporario e deletado depois.
		
		t.Run("dev com VERSION file adjacente: retorna versao-dev", func(t *testing.T) {
			Version = "dev"
			content := "0.11.2"
			err := os.WriteFile(versionFile, []byte(content), 0644)
			if err != nil {
				t.Skip("nao foi possivel criar VERSION adjacente ao executavel (permissao ou dir read-only)")
			}
			defer func() { _ = os.Remove(versionFile) }()

			got := ResolveFromExecutable()
			if got != content+"-dev" {
				t.Errorf("ResolveFromExecutable: got %q, want %q", got, content+"-dev")
			}
		})
		
		t.Run("dev sem VERSION file: retorna dev", func(t *testing.T) {
			Version = "dev"
			_ = os.Remove(versionFile) // garante que nao existe
			got := ResolveFromExecutable()
			if got != "dev" {
				t.Errorf("ResolveFromExecutable: got %q, want %q", got, "dev")
			}
		})
	})
}
