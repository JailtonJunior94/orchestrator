package detect

import (
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// FrameworkDetector detecta frameworks usados no projeto.
type FrameworkDetector struct {
	fs fs.FileSystem
}

func NewFrameworkDetector(fsys fs.FileSystem) *FrameworkDetector {
	return &FrameworkDetector{fs: fsys}
}

func (d *FrameworkDetector) Detect(projectDir string) []string {
	seen := make(map[string]bool)
	var frameworks []string

	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			frameworks = append(frameworks, name)
		}
	}

	// Go frameworks
	goMods := findManifestsRecursive(d.fs, projectDir, "go.mod", 4)
	for _, goMod := range goMods {
		data, err := d.fs.ReadFile(goMod)
		if err != nil {
			continue
		}
		content := string(data)

		if strings.Contains(content, "github.com/gin-gonic/gin") {
			add("Gin")
		}
		if strings.Contains(content, "github.com/labstack/echo") {
			add("Echo")
		}
		if strings.Contains(content, "github.com/gofiber/fiber") {
			add("Fiber")
		}
		if strings.Contains(content, "google.golang.org/grpc") {
			add("gRPC")
		}
		if strings.Contains(content, "connectrpc.com/connect") {
			add("Connect")
		}
	}

	// Node frameworks
	packages := findManifestsRecursive(d.fs, projectDir, "package.json", 4)
	for _, pkg := range packages {
		data, err := d.fs.ReadFile(pkg)
		if err != nil {
			continue
		}
		content := string(data)

		if strings.Contains(content, `"express"`) {
			add("Express")
		}
		if strings.Contains(content, `"@nestjs/core"`) {
			add("NestJS")
		}
		if strings.Contains(content, `"fastify"`) {
			add("Fastify")
		}
		if strings.Contains(content, `"next"`) {
			add("Next.js")
		}
		if strings.Contains(content, `"hono"`) {
			add("Hono")
		}
	}

	// Python frameworks
	pyprojects := findManifestsRecursive(d.fs, projectDir, "pyproject.toml", 4)
	for _, pyp := range pyprojects {
		data, err := d.fs.ReadFile(pyp)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))

		if strings.Contains(content, "fastapi") {
			add("FastAPI")
		}
		if strings.Contains(content, "django") {
			add("Django")
		}
		if strings.Contains(content, "flask") {
			add("Flask")
		}
	}

	requirements := findManifestsRecursive(d.fs, projectDir, "requirements.txt", 4)
	for _, req := range requirements {
		data, err := d.fs.ReadFile(req)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))

		if strings.Contains(content, "fastapi") {
			add("FastAPI")
		}
		if strings.Contains(content, "django") {
			add("Django")
		}
		if strings.Contains(content, "flask") {
			add("Flask")
		}
	}

	return frameworks
}

// DetectPrimaryStack detecta as stacks principais do projeto.
func DetectPrimaryStack(fsys fs.FileSystem, projectDir string) []string {
	var parts []string

	goIndicators := []string{"go.mod", "go.work"}
	for _, f := range goIndicators {
		if fsys.Exists(filepath.Join(projectDir, f)) {
			parts = append(parts, "Go")
			break
		}
	}

	nodeIndicators := []string{"package.json", "tsconfig.json"}
	for _, f := range nodeIndicators {
		if fsys.Exists(filepath.Join(projectDir, f)) {
			parts = append(parts, "Node.js")
			break
		}
	}

	pythonIndicators := []string{"pyproject.toml", "requirements.txt", "setup.py", "Pipfile"}
	for _, f := range pythonIndicators {
		if fsys.Exists(filepath.Join(projectDir, f)) {
			parts = append(parts, "Python")
			break
		}
	}

	if fsys.Exists(filepath.Join(projectDir, "pom.xml")) ||
		fsys.Exists(filepath.Join(projectDir, "build.gradle")) ||
		fsys.Exists(filepath.Join(projectDir, "build.gradle.kts")) {
		parts = append(parts, "Java/Kotlin")
	}

	if fsys.Exists(filepath.Join(projectDir, "Cargo.toml")) {
		parts = append(parts, "Rust")
	}

	if len(parts) == 0 {
		return []string{"stack principal nao detectada automaticamente"}
	}
	return parts
}

// JoinFrameworks junta frameworks com virgula ou retorna fallback.
func JoinFrameworks(frameworks []string) string {
	if len(frameworks) == 0 {
		return "nenhum framework dominante identificado"
	}
	return strings.Join(frameworks, ", ")
}
