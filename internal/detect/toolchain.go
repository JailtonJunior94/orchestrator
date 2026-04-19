package detect

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ToolchainEntry representa os comandos detectados para uma linguagem.
type ToolchainEntry struct {
	Fmt  string `json:"fmt"`
	Test string `json:"test"`
	Lint string `json:"lint"`
}

// ToolchainResult armazena o resultado da deteccao de toolchain por linguagem.
type ToolchainResult map[string]ToolchainEntry

// ToolchainDetector detecta comandos de fmt, test e lint de um projeto.
type ToolchainDetector struct {
	fs         fs.FileSystem
	maxDepth   int
	strict     bool
	warnWriter io.Writer
}

func NewToolchainDetector(fsys fs.FileSystem) *ToolchainDetector {
	return &ToolchainDetector{fs: fsys, maxDepth: 4}
}

// NewToolchainDetectorStrict cria um detector com modo strict habilitado.
// Warnings de binarios ausentes sao escritos em w.
func NewToolchainDetectorStrict(fsys fs.FileSystem, w io.Writer) *ToolchainDetector {
	return &ToolchainDetector{fs: fsys, maxDepth: 4, strict: true, warnWriter: w}
}

func (d *ToolchainDetector) Detect(projectDir string) ToolchainResult {
	result := make(ToolchainResult)

	if d.detectGo(projectDir) {
		result["go"] = ToolchainEntry{
			Fmt:  "gofmt -w .",
			Test: "go test ./...",
			Lint: "golangci-lint run",
		}
	}

	if entry, ok := d.detectNode(projectDir); ok {
		result["node"] = entry
	}

	if entry, ok := d.detectPython(projectDir); ok {
		result["python"] = entry
	}

	// Makefile fallback: quando nenhuma linguagem e detectada
	if len(result) == 0 {
		if entry, ok := d.detectMakefileFallback(projectDir); ok {
			result["unknown"] = entry
		}
	}

	if d.strict {
		for lang, entry := range result {
			d.warnMissingBinary(entry.Fmt, lang+"/fmt")
			d.warnMissingBinary(entry.Lint, lang+"/lint")
			d.warnMissingBinary(entry.Test, lang+"/test")
		}
	}

	return result
}

// warnMissingBinary verifica se o binario do comando esta no PATH e emite warning se ausente.
func (d *ToolchainDetector) warnMissingBinary(command, label string) {
	if command == "" || d.warnWriter == nil {
		return
	}
	binary := strings.Fields(command)[0]
	if _, err := exec.LookPath(binary); err != nil {
		fmt.Fprintf(d.warnWriter, "WARNING: binario ausente no PATH: %q (referenciado por %s)\n", binary, label)
	}
}

func (d *ToolchainDetector) detectGo(projectDir string) bool {
	return d.fs.Exists(filepath.Join(projectDir, "go.mod")) ||
		d.fs.Exists(filepath.Join(projectDir, "go.work")) ||
		d.findManifest(projectDir, "go.mod")
}

func (d *ToolchainDetector) detectNode(projectDir string) (ToolchainEntry, bool) {
	packages := d.findManifests(projectDir, "package.json")
	if len(packages) == 0 {
		return ToolchainEntry{}, false
	}

	pm := d.detectPackageManager(projectDir)
	var entry ToolchainEntry

	for _, pkg := range packages {
		scripts := d.parsePackageScripts(pkg)
		pkgName := d.parsePackageName(pkg)
		pkgDir := d.relativeDir(projectDir, pkg)

		cmdPrefix := pm + " run"
		if pkgDir != "" {
			if pkgName != "" && pm == "pnpm" {
				cmdPrefix = "pnpm --filter " + pkgName + " run"
			} else if pkgName != "" && pm == "yarn" {
				cmdPrefix = "yarn workspace " + pkgName + " run"
			} else {
				cmdPrefix = "cd " + pkgDir + " && " + pm + " run"
			}
		}

		if entry.Fmt == "" {
			if scripts["fmt"] {
				entry.Fmt = cmdPrefix + " fmt"
			} else if scripts["format"] {
				entry.Fmt = cmdPrefix + " format"
			}
		}
		if entry.Test == "" {
			if scripts["test"] {
				entry.Test = cmdPrefix + " test"
			} else if scripts["test:unit"] {
				entry.Test = cmdPrefix + " test:unit"
			}
		}
		if entry.Lint == "" && scripts["lint"] {
			entry.Lint = cmdPrefix + " lint"
		}
	}

	return entry, true
}

func (d *ToolchainDetector) detectPython(projectDir string) (ToolchainEntry, bool) {
	pyprojects := d.findManifests(projectDir, "pyproject.toml")
	requirements := d.findManifests(projectDir, "requirements.txt")
	hasSetupPy := d.fs.Exists(filepath.Join(projectDir, "setup.py"))
	hasPipfile := d.fs.Exists(filepath.Join(projectDir, "Pipfile"))

	if len(pyprojects) == 0 && len(requirements) == 0 && !hasSetupPy && !hasPipfile {
		return ToolchainEntry{}, false
	}

	var entry ToolchainEntry

	for _, manifest := range pyprojects {
		data, err := d.fs.ReadFile(manifest)
		if err != nil {
			continue
		}
		content := string(data)

		if entry.Fmt == "" || entry.Lint == "" {
			if strings.Contains(content, "[tool.ruff") || d.hasPyDep(content, "ruff") {
				if entry.Fmt == "" {
					entry.Fmt = "ruff format ."
				}
				if entry.Lint == "" {
					entry.Lint = "ruff check ."
				}
			}
		}
		if entry.Test == "" {
			if strings.Contains(content, "[tool.pytest") || d.hasPyDep(content, "pytest") {
				entry.Test = "pytest"
			}
		}
	}

	if entry.Test == "" {
		testsDir := filepath.Join(projectDir, "tests")
		if d.fs.IsDir(testsDir) || d.fs.Exists(filepath.Join(projectDir, "pytest.ini")) {
			entry.Test = "pytest"
		}
	}

	return entry, true
}

func (d *ToolchainDetector) detectPackageManager(projectDir string) string {
	if d.fs.Exists(filepath.Join(projectDir, "pnpm-lock.yaml")) ||
		d.fs.Exists(filepath.Join(projectDir, "pnpm-workspace.yaml")) {
		return "pnpm"
	}
	if d.fs.Exists(filepath.Join(projectDir, "yarn.lock")) {
		return "yarn"
	}
	if d.fs.Exists(filepath.Join(projectDir, "bun.lockb")) {
		return "bun"
	}
	return "npm"
}

func (d *ToolchainDetector) parsePackageScripts(pkgPath string) map[string]bool {
	scripts := make(map[string]bool)
	data, err := d.fs.ReadFile(pkgPath)
	if err != nil {
		return scripts
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return scripts
	}

	for k := range pkg.Scripts {
		scripts[k] = true
	}
	return scripts
}

func (d *ToolchainDetector) parsePackageName(pkgPath string) string {
	data, err := d.fs.ReadFile(pkgPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Name
}

func (d *ToolchainDetector) relativeDir(projectDir, manifestPath string) string {
	dir := filepath.Dir(manifestPath)
	rel, err := filepath.Rel(projectDir, dir)
	if err != nil || rel == "." {
		return ""
	}
	return rel
}

func (d *ToolchainDetector) hasPyDep(content, name string) bool {
	return strings.Contains(strings.ToLower(content), "\""+name)
}

func (d *ToolchainDetector) detectMakefileFallback(projectDir string) (ToolchainEntry, bool) {
	makefilePath := filepath.Join(projectDir, "Makefile")
	if !d.fs.Exists(makefilePath) {
		return ToolchainEntry{}, false
	}
	data, err := d.fs.ReadFile(makefilePath)
	if err != nil {
		return ToolchainEntry{}, false
	}
	targets := parseMakefileTargets(string(data))
	var entry ToolchainEntry
	if targets["fmt"] {
		entry.Fmt = "make fmt"
	}
	if targets["test"] {
		entry.Test = "make test"
	}
	if targets["lint"] {
		entry.Lint = "make lint"
	}
	return entry, true
}

func parseMakefileTargets(content string) map[string]bool {
	targets := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		for _, name := range []string{"fmt", "test", "lint"} {
			if strings.HasPrefix(line, name+":") {
				targets[name] = true
			}
		}
	}
	return targets
}

func (d *ToolchainDetector) findManifest(projectDir, name string) bool {
	manifests := d.findManifests(projectDir, name)
	return len(manifests) > 0
}

func (d *ToolchainDetector) findManifests(projectDir, name string) []string {
	return findManifestsRecursive(d.fs, projectDir, name, d.maxDepth)
}

// findManifestsRecursive busca arquivos por nome recursivamente ate maxDepth.
func findManifestsRecursive(fsys fs.FileSystem, baseDir, name string, maxDepth int) []string {
	var results []string
	findManifestsHelper(fsys, baseDir, name, 0, maxDepth, &results)
	return results
}

func findManifestsHelper(fsys fs.FileSystem, dir, name string, depth, maxDepth int, results *[]string) {
	if depth > maxDepth {
		return
	}

	// Diretorios a ignorar
	ignoreDirs := map[string]bool{
		"node_modules": true, "vendor": true, "dist": true,
		"build": true, "__pycache__": true, ".git": true,
	}

	target := filepath.Join(dir, name)
	if fsys.Exists(target) && !fsys.IsDir(target) {
		*results = append(*results, target)
	}

	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if ignoreDirs[e.Name()] {
			continue
		}
		findManifestsHelper(fsys, filepath.Join(dir, e.Name()), name, depth+1, maxDepth, results)
	}
}

// ToJSON serializa o resultado para JSON.
func (r ToolchainResult) ToJSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}
