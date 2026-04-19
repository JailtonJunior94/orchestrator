package prerequisites

import (
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// FileCheck descreve um arquivo (ou conjunto de alternativas) a verificar.
type FileCheck struct {
	Label    string   // descrição legível
	Paths    []string // caminhos alternativos (qualquer um satisfaz)
	Optional bool     // se true, ausência gera warning, não falha
}

// Check agrupa os requisitos de uma skill.
type Check struct {
	Skill    string
	Required []FileCheck
	Optional []FileCheck
}

// registry mapeia skill -> Check conforme scripts/check-skill-prerequisites.sh.
var registry = map[string]Check{
	"go-implementation": {
		Skill: "go-implementation",
		Required: []FileCheck{
			{Label: "go.mod ou go.work", Paths: []string{"go.mod", "go.work"}},
		},
	},
	"node-implementation": {
		Skill: "node-implementation",
		Required: []FileCheck{
			{Label: "package.json", Paths: []string{"package.json"}},
		},
	},
	"python-implementation": {
		Skill: "python-implementation",
		Required: []FileCheck{
			{Label: "pyproject.toml, setup.py ou requirements.txt", Paths: []string{"pyproject.toml", "setup.py", "requirements.txt"}},
		},
	},
	"create-tasks": {
		Skill: "create-tasks",
		Required: []FileCheck{
			{Label: "prd.md", Paths: []string{"prd.md"}},
			{Label: "techspec.md", Paths: []string{"techspec.md"}},
		},
	},
	"execute-task": {
		Skill: "execute-task",
		Required: []FileCheck{
			{Label: "tasks.md", Paths: []string{"tasks.md"}},
		},
	},
	"create-technical-specification": {
		Skill: "create-technical-specification",
		Required: []FileCheck{
			{Label: "prd.md", Paths: []string{"prd.md"}},
		},
	},
	"bugfix": {
		Skill: "bugfix",
		Optional: []FileCheck{
			{Label: "bugs.json", Paths: []string{"bugs.json"}, Optional: true},
		},
	},
}

// Result representa o resultado de verificação de um FileCheck.
type Result struct {
	Label   string
	Found   bool
	Optional bool
}

// Verify verifica os pré-requisitos de uma skill no diretório informado.
// Retorna:
//   - passed: true se todos os required foram encontrados
//   - results: lista de resultados por check
//   - err: skill desconhecida ou diretório inválido
func Verify(skill string, projectDir string, fsys fs.FileSystem) (passed bool, results []Result, err error) {
	check, known := registry[skill]
	if !known {
		return false, nil, fmt.Errorf("skill desconhecida: %q — skills suportadas: go-implementation, node-implementation, python-implementation, create-tasks, execute-task, create-technical-specification, bugfix", skill)
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return false, nil, fmt.Errorf("caminho invalido %q: %w", projectDir, err)
	}
	if !fsys.IsDir(absDir) {
		return false, nil, fmt.Errorf("diretorio nao encontrado: %s", absDir)
	}

	allPassed := true

	for _, fc := range check.Required {
		found := anyExists(fsys, absDir, fc.Paths)
		if !found {
			allPassed = false
		}
		results = append(results, Result{Label: fc.Label, Found: found, Optional: false})
	}

	for _, fc := range check.Optional {
		found := anyExists(fsys, absDir, fc.Paths)
		results = append(results, Result{Label: fc.Label, Found: found, Optional: true})
	}

	return allPassed, results, nil
}

// KnownSkills retorna a lista de skills registradas.
func KnownSkills() []string {
	skills := make([]string, 0, len(registry))
	for k := range registry {
		skills = append(skills, k)
	}
	return skills
}

func anyExists(fsys fs.FileSystem, dir string, paths []string) bool {
	for _, p := range paths {
		if fsys.Exists(filepath.Join(dir, p)) {
			return true
		}
	}
	return false
}
