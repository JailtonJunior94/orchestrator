package aispecharness

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeSpecDriftFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escrever fixture %s: %v", name, err)
	}
	return path
}

func specDriftHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func runCheckSpecDrift(t *testing.T, arg string) int {
	t.Helper()
	cmd := newCheckSpecDriftCmd()
	err := cmd.RunE(cmd, []string{arg})
	if err == nil {
		return 0
	}
	return NewExitResolver().CodeFor(err)
}

func TestCheckSpecDriftCmd_AceitaDiretorioEArquivo(t *testing.T) {
	cases := []struct {
		name     string
		prepare  func(t *testing.T, root string) string
		wantCode int
	}{
		{
			name: "diretorio prd sem drift retorna zero",
			prepare: func(t *testing.T, root string) string {
				prd := "RF-01 deve existir"
				techspec := "REQ-01 deve ser implementado"
				tasks := fmt.Sprintf("RF-01 coberto. REQ-01 coberto.\n<!-- spec-hash-prd: %s -->\n<!-- spec-hash-techspec: %s -->\n", specDriftHash(prd), specDriftHash(techspec))
				writeSpecDriftFixture(t, root, "prd.md", prd)
				writeSpecDriftFixture(t, root, "techspec.md", techspec)
				writeSpecDriftFixture(t, root, "tasks.md", tasks)
				return root
			},
			wantCode: 0,
		},
		{
			name: "diretorio prd com rf descoberto retorna drift",
			prepare: func(t *testing.T, root string) string {
				prd := "RF-01 e RF-02 devem existir"
				tasks := fmt.Sprintf("RF-01 coberto.\n<!-- spec-hash-prd: %s -->\n", specDriftHash(prd))
				writeSpecDriftFixture(t, root, "prd.md", prd)
				writeSpecDriftFixture(t, root, "tasks.md", tasks)
				return root
			},
			wantCode: 1,
		},
		{
			name: "arquivo tasks md preserva comportamento anterior",
			prepare: func(t *testing.T, root string) string {
				prd := "RF-01 deve existir"
				tasks := fmt.Sprintf("RF-01 coberto.\n<!-- spec-hash-prd: %s -->\n", specDriftHash(prd))
				writeSpecDriftFixture(t, root, "prd.md", prd)
				return writeSpecDriftFixture(t, root, "tasks.md", tasks)
			},
			wantCode: 0,
		},
		{
			name: "caminho inexistente retorna erro de uso",
			prepare: func(t *testing.T, root string) string {
				return filepath.Join(root, "inexistente")
			},
			wantCode: 2,
		},
		{
			name: "diretorio sem tasks retorna erro de uso",
			prepare: func(t *testing.T, root string) string {
				writeSpecDriftFixture(t, root, "prd.md", "RF-01 deve existir")
				return root
			},
			wantCode: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			arg := tc.prepare(t, root)

			gotCode := runCheckSpecDrift(t, arg)
			if gotCode != tc.wantCode {
				t.Fatalf("codigo de saida = %d, quer %d", gotCode, tc.wantCode)
			}
		})
	}
}
