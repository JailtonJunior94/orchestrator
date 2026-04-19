package detect

// stack_coverage_test.go valida deteccao de linguagem, framework, toolchain e arquitetura
// para cada stack suportada (Go, Node.js, Python) usando fixtures reais em disco.
//
// Fixtures e o que cada uma valida:
//
//   testdata/go-monolith
//     - linguagem Go via go.mod
//     - toolchain Go: gofmt, go test, golangci-lint (hardcoded no detector)
//     - arquitetura monolito (sem Dockerfile/k8s/go.work)
//     - sem frameworks detectados (monolito puro)
//
//   testdata/node-api
//     - linguagem Node via package.json
//     - framework Express via campo dependencies
//     - toolchain npm run fmt/test/lint via scripts do package.json
//     - arquitetura monolito (sem workspace/apps/packages)
//
//   testdata/python-api
//     - linguagem Python via pyproject.toml
//     - framework FastAPI via dependencias declaradas
//     - toolchain ruff format/check + pytest via pyproject.toml
//     - arquitetura microservico (Dockerfile + k8s/deployment.yaml)
//
//   testdata/go-microservice
//     - linguagem Go via go.mod
//     - framework Gin via require no go.mod
//     - arquitetura microservico (Dockerfile + k8s/deployment.yaml)
//     - toolchain Go: gofmt, go test, golangci-lint

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestStackCoverage_Go_Monolith(t *testing.T) {
	dir := fixtureDir("go-monolith")
	osfs := fs.NewOSFileSystem()

	t.Run("language", func(t *testing.T) {
		det := NewFileDetector(osfs)
		langs := det.DetectLangs(dir)
		found := false
		for _, l := range langs {
			if l == skills.LangGo {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Go in langs, got %v", langs)
		}
	})

	t.Run("framework", func(t *testing.T) {
		// go-monolith nao usa frameworks externos; verifica que nenhum e detectado erroneamente
		det := NewFrameworkDetector(osfs)
		frameworks := det.Detect(dir)
		if len(frameworks) != 0 {
			t.Errorf("go-monolith nao deve ter frameworks detectados, got %v", frameworks)
		}
	})

	t.Run("toolchain", func(t *testing.T) {
		det := NewToolchainDetector(osfs)
		result := det.Detect(dir)
		entry, ok := result["go"]
		if !ok {
			t.Fatal("expected go toolchain entry")
		}
		if entry.Fmt != "gofmt -w ." {
			t.Errorf("fmt: got %q", entry.Fmt)
		}
		if entry.Test != "go test ./..." {
			t.Errorf("test: got %q", entry.Test)
		}
		if entry.Lint != "golangci-lint run" {
			t.Errorf("lint: got %q", entry.Lint)
		}
	})

	t.Run("architecture", func(t *testing.T) {
		det := NewArchitectureDetector(osfs)
		result := det.Detect(dir)
		if result.Type != ArchMonolith {
			t.Errorf("expected %s, got %s", ArchMonolith, result.Type)
		}
	})

	t.Run("primary_stack", func(t *testing.T) {
		stacks := DetectPrimaryStack(osfs, dir)
		found := false
		for _, s := range stacks {
			if s == "Go" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Go in primary stacks, got %v", stacks)
		}
	})
}

func TestStackCoverage_Node_API(t *testing.T) {
	dir := fixtureDir("node-api")
	osfs := fs.NewOSFileSystem()

	t.Run("language", func(t *testing.T) {
		det := NewFileDetector(osfs)
		langs := det.DetectLangs(dir)
		found := false
		for _, l := range langs {
			if l == skills.LangNode {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Node in langs, got %v", langs)
		}
	})

	t.Run("framework", func(t *testing.T) {
		det := NewFrameworkDetector(osfs)
		frameworks := det.Detect(dir)
		found := false
		for _, f := range frameworks {
			if f == "Express" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Express in frameworks, got %v", frameworks)
		}
	})

	t.Run("toolchain", func(t *testing.T) {
		det := NewToolchainDetector(osfs)
		result := det.Detect(dir)
		entry, ok := result["node"]
		if !ok {
			t.Fatal("expected node toolchain entry")
		}
		if entry.Fmt != "npm run fmt" {
			t.Errorf("fmt: got %q", entry.Fmt)
		}
		if entry.Test != "npm run test" {
			t.Errorf("test: got %q", entry.Test)
		}
		if entry.Lint != "npm run lint" {
			t.Errorf("lint: got %q", entry.Lint)
		}
	})

	t.Run("architecture", func(t *testing.T) {
		det := NewArchitectureDetector(osfs)
		result := det.Detect(dir)
		if result.Type != ArchMonolith {
			t.Errorf("expected %s, got %s", ArchMonolith, result.Type)
		}
	})

	t.Run("primary_stack", func(t *testing.T) {
		stacks := DetectPrimaryStack(osfs, dir)
		found := false
		for _, s := range stacks {
			if s == "Node.js" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Node.js in primary stacks, got %v", stacks)
		}
	})
}

func TestStackCoverage_Python_API(t *testing.T) {
	dir := fixtureDir("python-api")
	osfs := fs.NewOSFileSystem()

	t.Run("language", func(t *testing.T) {
		det := NewFileDetector(osfs)
		langs := det.DetectLangs(dir)
		found := false
		for _, l := range langs {
			if l == skills.LangPython {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Python in langs, got %v", langs)
		}
	})

	t.Run("framework", func(t *testing.T) {
		det := NewFrameworkDetector(osfs)
		frameworks := det.Detect(dir)
		found := false
		for _, f := range frameworks {
			if f == "FastAPI" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected FastAPI in frameworks, got %v", frameworks)
		}
	})

	t.Run("toolchain", func(t *testing.T) {
		det := NewToolchainDetector(osfs)
		result := det.Detect(dir)
		entry, ok := result["python"]
		if !ok {
			t.Fatal("expected python toolchain entry")
		}
		if entry.Fmt != "ruff format ." {
			t.Errorf("fmt: got %q", entry.Fmt)
		}
		if entry.Test != "pytest" {
			t.Errorf("test: got %q", entry.Test)
		}
		if entry.Lint != "ruff check ." {
			t.Errorf("lint: got %q", entry.Lint)
		}
	})

	t.Run("architecture", func(t *testing.T) {
		det := NewArchitectureDetector(osfs)
		result := det.Detect(dir)
		if result.Type != ArchMicroservice {
			t.Errorf("expected %s, got %s", ArchMicroservice, result.Type)
		}
	})

	t.Run("primary_stack", func(t *testing.T) {
		stacks := DetectPrimaryStack(osfs, dir)
		found := false
		for _, s := range stacks {
			if s == "Python" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Python in primary stacks, got %v", stacks)
		}
	})
}

// TestStackCoverage_Go_Microservice valida deteccao de framework (Gin) via go.mod
// e arquitetura microservico (Dockerfile + k8s) para o stack Go com container.
func TestStackCoverage_Go_Microservice(t *testing.T) {
	dir := fixtureDir("go-microservice")
	osfs := fs.NewOSFileSystem()

	t.Run("language", func(t *testing.T) {
		det := NewFileDetector(osfs)
		langs := det.DetectLangs(dir)
		found := false
		for _, l := range langs {
			if l == skills.LangGo {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Go in langs, got %v", langs)
		}
	})

	t.Run("framework", func(t *testing.T) {
		// go.mod declara github.com/gin-gonic/gin; detector deve reconhecer Gin
		det := NewFrameworkDetector(osfs)
		frameworks := det.Detect(dir)
		found := false
		for _, f := range frameworks {
			if f == "Gin" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Gin in frameworks, got %v", frameworks)
		}
	})

	t.Run("toolchain", func(t *testing.T) {
		det := NewToolchainDetector(osfs)
		result := det.Detect(dir)
		entry, ok := result["go"]
		if !ok {
			t.Fatal("expected go toolchain entry")
		}
		if entry.Fmt != "gofmt -w ." {
			t.Errorf("fmt: got %q", entry.Fmt)
		}
		if entry.Test != "go test ./..." {
			t.Errorf("test: got %q", entry.Test)
		}
		if entry.Lint != "golangci-lint run" {
			t.Errorf("lint: got %q", entry.Lint)
		}
	})

	t.Run("architecture", func(t *testing.T) {
		// Dockerfile + k8s/deployment.yaml definem arquitetura microservico
		det := NewArchitectureDetector(osfs)
		result := det.Detect(dir)
		if result.Type != ArchMicroservice {
			t.Errorf("expected %s, got %s", ArchMicroservice, result.Type)
		}
	})

	t.Run("primary_stack", func(t *testing.T) {
		stacks := DetectPrimaryStack(osfs, dir)
		found := false
		for _, s := range stacks {
			if s == "Go" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Go in primary stacks, got %v", stacks)
		}
	})
}
