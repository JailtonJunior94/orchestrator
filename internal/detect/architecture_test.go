package detect

import (
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func fixtureDir(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestDetectArchitecture_Monorepo_GoWork(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.work"] = []byte("go 1.23")
	ffs.Files["/project/services/api/go.mod"] = []byte("module api")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchMonorepo {
		t.Errorf("expected monorepo, got %s", result.Type)
	}
}

func TestDetectArchitecture_Monorepo_PnpmWorkspace(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/pnpm-workspace.yaml"] = []byte("packages: ['apps/*']")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchMonorepo {
		t.Errorf("expected monorepo, got %s", result.Type)
	}
}

func TestDetectArchitecture_Monorepo_AppsPackages(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/apps/web/index.ts"] = []byte("")
	ffs.Files["/project/packages/shared/index.ts"] = []byte("")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchMonorepo {
		t.Errorf("expected monorepo, got %s", result.Type)
	}
}

func TestDetectArchitecture_Modular(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/internal/order/service.go"] = []byte("")
	ffs.Files["/project/internal/customer/service.go"] = []byte("")
	ffs.Files["/project/internal/payment/service.go"] = []byte("")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchModular {
		t.Errorf("expected monolito modular, got %s", result.Type)
	}
}

func TestDetectArchitecture_Microservice(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/Dockerfile"] = []byte("FROM golang")
	ffs.Files["/project/k8s/deployment.yaml"] = []byte("apiVersion: apps/v1")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchMicroservice {
		t.Errorf("expected microservico, got %s", result.Type)
	}
}

func TestDetectArchitecture_Monolith_Fallback(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/main.go"] = []byte("package main")
	ffs.Dirs["/project"] = true

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Type != ArchMonolith {
		t.Errorf("expected monolito, got %s", result.Type)
	}
}

func TestDetectArchitecturalPattern_CleanArch(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/domain/user.go"] = []byte("")
	ffs.Files["/project/application/service.go"] = []byte("")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	if result.Pattern == "" {
		t.Error("pattern should not be empty")
	}
	if result.Pattern != "Predominio de Clean Architecture / Hexagonal com fronteiras explicitas entre dominio, aplicacao e infraestrutura." {
		t.Errorf("unexpected pattern: %s", result.Pattern)
	}
}

func TestDetectArchitecturalPattern_Layered(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/controllers/handler.go"] = []byte("")
	ffs.Files["/project/services/user.go"] = []byte("")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	expected := "Predominio de arquitetura em camadas, com separacao entre transporte, servicos, persistencia e modelos."
	if result.Pattern != expected {
		t.Errorf("expected %q, got %q", expected, result.Pattern)
	}
}

func TestDetectArchitecturalPattern_Internal(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/internal/order/service.go"] = []byte("")

	det := NewArchitectureDetector(ffs)
	result := det.Detect("/project")

	expected := "Predominio de packages internos coesos, com estrutura orientada por dominio ou componente."
	if result.Pattern != expected {
		t.Errorf("expected %q, got %q", expected, result.Pattern)
	}
}

func TestDetectArchitecture_Fixture_GoMicroservice(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("go-microservice"))

	if result.Type != ArchMicroservice {
		t.Errorf("expected %s, got %s", ArchMicroservice, result.Type)
	}
}

func TestDetectArchitecture_Fixture_GoModular(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("go-modular"))

	if result.Type != ArchModular {
		t.Errorf("expected %s, got %s", ArchModular, result.Type)
	}
}

func TestDetectArchitecture_Fixture_NodeMonorepo(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("node-monorepo"))

	if result.Type != ArchMonorepo {
		t.Errorf("expected %s, got %s", ArchMonorepo, result.Type)
	}
}

func TestDetectArchitecture_Fixture_PolyglotMonorepo(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("polyglot-monorepo"))

	if result.Type != ArchMonorepo {
		t.Errorf("expected %s, got %s", ArchMonorepo, result.Type)
	}
}

func TestDetectArchitecture_Fixture_PythonAPI(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("python-api"))

	if result.Type != ArchMicroservice {
		t.Errorf("expected %s, got %s", ArchMicroservice, result.Type)
	}
}

func TestDetectArchitecture_Fixture_GoMonolith(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("go-monolith"))

	if result.Type != ArchMonolith {
		t.Errorf("expected %s, got %s", ArchMonolith, result.Type)
	}
}

func TestDetectArchitecture_Fixture_NodeAPI(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("node-api"))

	if result.Type != ArchMonolith {
		t.Errorf("expected %s, got %s", ArchMonolith, result.Type)
	}
}

func TestDetectArchitecture_Fixture_PythonMonorepo(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	det := NewArchitectureDetector(osfs)
	result := det.Detect(fixtureDir("python-monorepo"))

	if result.Type != ArchMonorepo {
		t.Errorf("expected %s, got %s", ArchMonorepo, result.Type)
	}
}
