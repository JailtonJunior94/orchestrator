package taskloop

import (
	"testing"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// TestResolveWorkDir valida a logica de busca da raiz do projeto via marcadores.
func TestResolveWorkDir(t *testing.T) {
	tests := []struct {
		name      string
		prdFolder string
		setup     func(fsys *taskfs.FakeFileSystem)
		want      string
	}{
		{
			name:      ".git presente no diretorio pai",
			prdFolder: "/fake/project/tasks/prd-feature",
			setup: func(fsys *taskfs.FakeFileSystem) {
				// FakeFileSystem.Exists retorna true para prefixo de arquivo existente.
				fsys.Files["/fake/project/.git/HEAD"] = []byte("ref: refs/heads/main")
			},
			want: "/fake/project",
		},
		{
			name:      "go.mod presente no diretorio corrente",
			prdFolder: "/fake/project",
			setup: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files["/fake/project/go.mod"] = []byte("module example.com/app\n")
			},
			want: "/fake/project",
		},
		{
			name:      "AGENTS.md presente em diretorio ancestral",
			prdFolder: "/fake/project/tasks/prd-feature",
			setup: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files["/fake/project/AGENTS.md"] = []byte("# Agents\n")
			},
			want: "/fake/project",
		},
		{
			name:      "nenhum marker encontrado — fallback para prdFolder",
			prdFolder: "/fake/isolated/prd-feature",
			setup:     func(fsys *taskfs.FakeFileSystem) {},
			want:      "/fake/isolated/prd-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := taskfs.NewFakeFileSystem()
			tt.setup(fsys)

			got, err := resolveWorkDir(tt.prdFolder, fsys)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveWorkDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
