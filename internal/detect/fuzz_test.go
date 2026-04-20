package detect

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// FuzzDetectLanguages verifica que deteccao de linguagens com inputs malformados
// no filesystem nao causa panic. O detector e exposto a caminhos arbitrarios
// e nomes de arquivo inesperados.
func FuzzDetectLanguages(f *testing.F) {
	// Corpus: caminhos normais de indicadores de linguagem
	f.Add("go.mod")
	f.Add("package.json")
	f.Add("pyproject.toml")
	f.Add("requirements.txt")
	f.Add("")
	f.Add("../../../etc/passwd")
	f.Add("go.mod\x00")
	f.Add(string(make([]byte, 1024)))

	f.Fuzz(func(t *testing.T, indicator string) {
		fake := fs.NewFakeFileSystem()
		// Criar um arquivo com o nome fuzzeado — nao deve causar panic
		if indicator != "" {
			_ = fake.WriteFile("/project/"+indicator, []byte("content"))
		}
		det := NewFileDetector(fake)
		// Detectar linguagens nao deve causar panic independente do filesystem
		_ = det.DetectLangs("/project")
	})
}

// FuzzDetectToolchain verifica que deteccao de toolchain com conteudo arbitrario
// nos arquivos de manifesto nao causa panic.
func FuzzDetectToolchain(f *testing.F) {
	// Corpus: conteudo valido de go.mod
	f.Add("module example.com/myapp\n\ngo 1.21\n")

	// Corpus: conteudo valido de package.json
	f.Add(`{"name":"myapp","version":"1.0.0","scripts":{"test":"jest"}}`)

	// Corpus: conteudo vazio
	f.Add("")

	// Corpus: binario arbitrario
	f.Add(string([]byte{0xff, 0xfe, 0x00, 0x01, 0x02}))

	// Corpus: texto muito longo
	f.Add(string(make([]byte, 4096)))

	// Corpus: JSON invalido
	f.Add("{not valid json}")

	f.Fuzz(func(t *testing.T, fileContent string) {
		fake := fs.NewFakeFileSystem()
		// Criar go.mod com conteudo fuzzeado
		_ = fake.WriteFile("/project/go.mod", []byte(fileContent))
		det := NewToolchainDetector(fake)
		// Detectar toolchain nao deve causar panic
		_ = det.Detect("/project")
	})
}
