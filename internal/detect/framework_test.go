package detect

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestFrameworkDetect_Go(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte(`module example

require (
	github.com/gin-gonic/gin v1.10.0
	google.golang.org/grpc v1.65.0
)`)

	det := NewFrameworkDetector(ffs)
	frameworks := det.Detect("/project")

	if len(frameworks) != 2 {
		t.Fatalf("expected 2 frameworks, got %d: %v", len(frameworks), frameworks)
	}
	if frameworks[0] != "Gin" {
		t.Errorf("expected Gin, got %s", frameworks[0])
	}
	if frameworks[1] != "gRPC" {
		t.Errorf("expected gRPC, got %s", frameworks[1])
	}
}

func TestFrameworkDetect_Node(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/package.json"] = []byte(`{
		"dependencies": {
			"express": "^4.0.0",
			"next": "^14.0.0"
		}
	}`)

	det := NewFrameworkDetector(ffs)
	frameworks := det.Detect("/project")

	if len(frameworks) != 2 {
		t.Fatalf("expected 2 frameworks, got %d: %v", len(frameworks), frameworks)
	}
}

func TestFrameworkDetect_Python(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/pyproject.toml"] = []byte(`[project]
dependencies = ["fastapi>=0.111"]`)

	det := NewFrameworkDetector(ffs)
	frameworks := det.Detect("/project")

	if len(frameworks) != 1 || frameworks[0] != "FastAPI" {
		t.Errorf("expected [FastAPI], got %v", frameworks)
	}
}

func TestFrameworkDetect_Empty(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	det := NewFrameworkDetector(ffs)
	frameworks := det.Detect("/project")

	if len(frameworks) != 0 {
		t.Errorf("expected 0 frameworks, got %v", frameworks)
	}
}

func TestFrameworkDetect_Deduplication(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("require github.com/gin-gonic/gin v1.10.0")
	ffs.Files["/project/services/api/go.mod"] = []byte("require github.com/gin-gonic/gin v1.10.0")

	det := NewFrameworkDetector(ffs)
	frameworks := det.Detect("/project")

	if len(frameworks) != 1 {
		t.Errorf("expected 1 framework (deduplicated), got %d: %v", len(frameworks), frameworks)
	}
}

func TestJoinFrameworks(t *testing.T) {
	if got := JoinFrameworks(nil); got != "nenhum framework dominante identificado" {
		t.Errorf("empty: got %q", got)
	}
	if got := JoinFrameworks([]string{"Gin", "gRPC"}); got != "Gin, gRPC" {
		t.Errorf("two: got %q", got)
	}
}

func TestDetectPrimaryStack(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")
	ffs.Files["/project/package.json"] = []byte("{}")

	stacks := DetectPrimaryStack(ffs, "/project")
	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d: %v", len(stacks), stacks)
	}
	if stacks[0] != "Go" || stacks[1] != "Node.js" {
		t.Errorf("unexpected stacks: %v", stacks)
	}
}
