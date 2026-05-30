package lint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	internalfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
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

// _targetFiles são os arquivos verificados por placeholders não renderizados.
var _targetFiles = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"GEMINI.md",
	".codex/config.toml",
	".github/copilot-instructions.md",
}

var _schemaVersionRe = regexp.MustCompile(`governance-schema:\s*(\S+)`)

// Service executa verificações de lint de governança.
type Service struct {
	fs internalfs.FileSystem
}

// NewService cria um novo Service de lint.
func NewService(fsys ...internalfs.FileSystem) *Service {
	if len(fsys) > 0 && fsys[0] != nil {
		return &Service{fs: fsys[0]}
	}
	return &Service{fs: internalfs.NewOSFileSystem()}
}

// Execute executa o lint no projectDir e retorna a lista de erros encontrados.
// Retorna nil, nil quando não há erros.
func (s *Service) Execute(projectDir string) ([]LintError, error) {
	var errs []LintError

	// 1. Detectar placeholders {{ em arquivos alvo
	for _, rel := range _targetFiles {
		path := filepath.Join(projectDir, rel)
		fileErrs, err := s.checkPlaceholders(path, rel)
		if err != nil {
			continue // arquivo não existe — pular
		}
		errs = append(errs, fileErrs...)
	}

	// 2. Verificar governance-schema em AGENTS.md
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	data, err := s.fs.ReadFile(agentsPath)
	if err == nil {
		if vErr := s.checkSchemaVersion(data, "AGENTS.md"); vErr != nil {
			errs = append(errs, *vErr)
		}
	}

	// 3. Validar bug-schema.json como JSON válido
	bugSchemaPath := filepath.Join(projectDir, ".agents", "skills", "agent-governance", "references", "bug-schema.json")
	bugData, err := s.fs.ReadFile(bugSchemaPath)
	if err == nil {
		if !json.Valid(bugData) {
			errs = append(errs, LintError{
				File:    bugSchemaPath,
				Message: "bug-schema.json nao e JSON valido",
			})
		}
	}

	// 4. Validar frontmatter dos SKILL.md
	skillErrs := s.checkSkillFrontmatters(projectDir)
	errs = append(errs, skillErrs...)

	if len(errs) == 0 {
		return nil, nil
	}
	return errs, nil
}

// CountChecks retorna o número de verificações que Execute realizaria no projectDir.
func (s *Service) CountChecks(projectDir string) int {
	count := 0

	for _, rel := range _targetFiles {
		if s.fs.Exists(filepath.Join(projectDir, rel)) {
			count++
		}
	}

	// schema version check (usa AGENTS.md — já contado acima se presente)
	if s.fs.Exists(filepath.Join(projectDir, "AGENTS.md")) {
		count++ // conta separadamente pois é uma verificação distinta
	}

	if s.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "agent-governance", "references", "bug-schema.json")) {
		count++
	}

	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	entries, err := s.fs.ReadDir(skillsDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
				if s.fs.Exists(skillFile) {
					count++
				}
			}
		}
	}

	return count
}

func (s *Service) checkPlaceholders(path, rel string) ([]LintError, error) {
	data, err := s.fs.ReadFile(path)
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
				Message: fmt.Sprintf("placeholder nao renderizado: %s", s.extractPlaceholder(line)),
			})
		}
	}
	return errs, nil
}

func (s *Service) extractPlaceholder(line string) string {
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

func (s *Service) checkSchemaVersion(data []byte, rel string) *LintError {
	matches := _schemaVersionRe.FindSubmatch(data)
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

func (s *Service) checkSkillFrontmatters(projectDir string) []LintError {
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	entries, err := s.fs.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	thirdPartySkills := s.thirdPartySkills(projectDir)
	catalog := skills.NewCatalog()
	var errs []LintError
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := s.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}
		if err := catalog.ValidateFrontmatter(data, "", nil); err != nil {
			errs = append(errs, LintError{
				File:    skillFile,
				Message: fmt.Sprintf("frontmatter invalido: %s", err),
			})
			continue
		}
		schemaData := s.frontmatterSchemaContent(data, thirdPartySkills[e.Name()])
		if err := catalog.ValidateFrontmatterSchema(schemaData, e.Name()); err != nil {
			errs = append(errs, LintError{
				File:    skillFile,
				Message: fmt.Sprintf("schema invalido: %s", err),
			})
		}
	}
	return errs
}

func (s *Service) thirdPartySkills(projectDir string) map[string]bool {
	lockPath := filepath.Join(projectDir, "skills-lock.json")
	lockData, err := s.fs.ReadFile(lockPath)
	if err != nil {
		return nil
	}

	var lock skillscheck.LockFile
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return nil
	}

	thirdParty := make(map[string]bool, len(lock.Skills))
	for skillName, entry := range lock.Skills {
		if s.isThirdPartySourceType(entry.SourceType) {
			thirdParty[skillName] = true
		}
	}
	return thirdParty
}

func (s *Service) isThirdPartySourceType(sourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "external", "git", "github", "registry", "agentskills":
		return true
	default:
		return false
	}
}

func (s *Service) frontmatterSchemaContent(content []byte, thirdParty bool) []byte {
	if !thirdParty || skills.NewCatalog().ParseFrontmatter(content).Version != "" {
		return content
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}

	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[0])
	inserted := false
	for _, line := range lines[1:] {
		if !inserted && strings.TrimSpace(line) == "---" {
			updated = append(updated, "version: 0.0.0")
			inserted = true
		}
		updated = append(updated, line)
	}
	if !inserted {
		return content
	}
	return []byte(strings.Join(updated, "\n"))
}
