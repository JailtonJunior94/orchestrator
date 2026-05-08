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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(SetForTest(tc.ldflags))
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
	t.Run("ldflags setado: retorna ldflags sem tocar filesystem", func(t *testing.T) {
		t.Cleanup(SetForTest("1.1.0-test"))
		got := ResolveFromExecutable()
		if got != "1.1.0-test" {
			t.Errorf("ResolveFromExecutable: got %q, want %q", got, "1.1.0-test")
		}
	})

	t.Run("dev com VERSION file adjacente: retorna versao-dev", func(t *testing.T) {
		t.Cleanup(SetForTest("dev"))
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		resolved, _ := filepath.EvalSymlinks(exe)
		dir := filepath.Dir(resolved)
		versionFile := filepath.Join(dir, "VERSION")

		content := "0.11.2"
		if err := os.WriteFile(versionFile, []byte(content), 0644); err != nil {
			t.Skip("nao foi possivel criar VERSION adjacente ao executavel (permissao ou dir read-only)")
		}
		defer func() { _ = os.Remove(versionFile) }()

		got := ResolveFromExecutable()
		if got != content+"-dev" {
			t.Errorf("ResolveFromExecutable: got %q, want %q", got, content+"-dev")
		}
	})

	t.Run("dev sem VERSION file: retorna dev", func(t *testing.T) {
		t.Cleanup(SetForTest("dev"))
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		resolved, _ := filepath.EvalSymlinks(exe)
		dir := filepath.Dir(resolved)
		versionFile := filepath.Join(dir, "VERSION")
		_ = os.Remove(versionFile) // garante que nao existe

		got := ResolveFromExecutable()
		if got != "dev" {
			t.Errorf("ResolveFromExecutable: got %q, want %q", got, "dev")
		}
	})
}

// TestSetForTest_Concurrent valida que SetForTest serializa escritores
// concorrentes e que leituras via Get permanecem consistentes (sem deadlock,
// sem data race quando a suite roda com -race).
func TestSetForTest_Concurrent(t *testing.T) {
	const N = 50
	done := make(chan struct{}, N)
	for i := 0; i < N; i++ {
		go func() {
			restore := SetForTest("concurrent-test")
			_ = Get()
			restore()
			done <- struct{}{}
		}()
	}
	for i := 0; i < N; i++ {
		<-done
	}
}
