package lint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// LintError representa um erro de lint com arquivo, linha e mensagem.
type LintError struct {
	File    string
	Line    int
	Message string
}

func (e LintError) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("ERRO: %s:%d: %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("ERRO: %s: %s", e.File, e.Message)
}

// targetFiles são os arquivos verificados por placeholders não renderizados.
var targetFiles = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"GEMINI.md",
	".codex/config.toml",
	".github/copilot-instructions.md",
}

var schemaVersionRe = regexp.MustCompile(`governance-schema:\s*(\S+)`)

// Service executa verificações de lint de governança.
type Service struct{}

// NewService cria um novo Service de lint.
func NewService() *Service {
	return &Service{}
}

// Execute executa o lint no projectDir e retorna a lista de erros encontrados.
// Retorna nil, nil quando não há erros.
func (s *Service) Execute(projectDir string) ([]LintError, error) {
	var errs []LintError

	// 1. Detectar placeholders {{ em arquivos alvo
	for _, rel := range targetFiles {
		path := filepath.Join(projectDir, rel)
		fileErrs, err := checkPlaceholders(path, rel)
		if err != nil {
			continue // arquivo não existe — pular
		}
		errs = append(errs, fileErrs...)
	}

	// 2. Verificar governance-schema em AGENTS.md
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err == nil {
		if vErr := checkSchemaVersion(data, "AGENTS.md"); vErr != nil {
			errs = append(errs, *vErr)
		}
	}

	// 3. Validar bug-schema.json como JSON válido
	bugSchemaPath := filepath.Join(projectDir, ".agents", "skills", "agent-governance", "references", "bug-schema.json")
	bugData, err := os.ReadFile(bugSchemaPath)
	if err == nil {
		if !json.Valid(bugData) {
			errs = append(errs, LintError{
				File:    bugSchemaPath,
				Message: "bug-schema.json nao e JSON valido",
			})
		}
	}

	// 4. Validar frontmatter dos SKILL.md
	skillErrs := checkSkillFrontmatters(projectDir)
	errs = append(errs, skillErrs...)

	if len(errs) == 0 {
		return nil, nil
	}
	return errs, nil
}

// CountChecks retorna o número de verificações que Execute realizaria no projectDir.
func (s *Service) CountChecks(projectDir string) int {
	count := 0

	for _, rel := range targetFiles {
		if _, err := os.Stat(filepath.Join(projectDir, rel)); err == nil {
			count++
		}
	}

	// schema version check (usa AGENTS.md — já contado acima se presente)
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err == nil {
		count++ // conta separadamente pois é uma verificação distinta
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "agent-governance", "references", "bug-schema.json")); err == nil {
		count++
	}

	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					count++
				}
			}
		}
	}

	return count
}

func checkPlaceholders(path, rel string) ([]LintError, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var errs []LintError
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, "{{") {
			errs = append(errs, LintError{
				File:    rel,
				Line:    lineNum,
				Message: fmt.Sprintf("placeholder nao renderizado: %s", extractPlaceholder(line)),
			})
		}
	}
	return errs, nil
}

func extractPlaceholder(line string) string {
	start := strings.Index(line, "{{")
	if start < 0 {
		return "{{"
	}
	end := strings.Index(line[start:], "}}")
	if end < 0 {
		return strings.TrimSpace(line[start:])
	}
	return strings.TrimSpace(line[start : start+end+2])
}

func checkSchemaVersion(data []byte, rel string) *LintError {
	matches := schemaVersionRe.FindSubmatch(data)
	if matches == nil {
		return nil
	}
	found := string(matches[1])
	expected := contextgen.GovernanceSchemaVersion
	if found != expected {
		return &LintError{
			File:    rel,
			Message: fmt.Sprintf("governance-schema %q diverge da versao atual %q", found, expected),
		}
	}
	return nil
}

func checkSkillFrontmatters(projectDir string) []LintError {
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var errs []LintError
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		if err := skills.ValidateFrontmatter(data, "", nil); err != nil {
			errs = append(errs, LintError{
				File:    skillFile,
				Message: fmt.Sprintf("frontmatter invalido: %s", err),
			})
			continue
		}
		if err := skills.ValidateFrontmatterSchema(data, e.Name()); err != nil {
			errs = append(errs, LintError{
				File:    skillFile,
				Message: fmt.Sprintf("schema invalido: %s", err),
			})
		}
	}
	return errs
}
