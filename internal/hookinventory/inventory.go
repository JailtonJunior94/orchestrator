package hookinventory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

type Entry struct {
	Name           string `json:"name"`
	Location       string `json:"location"`
	Event          string `json:"event"`
	Provider       string `json:"provider"`
	Purpose        string `json:"purpose"`
	PolicyGate     string `json:"policy_gate"`
	Blocking       string `json:"blocking"`
	Cost           string `json:"cost"`
	FailureMode    string `json:"failure_mode"`
	TestCoverage   string `json:"test_coverage"`
	Duplication    string `json:"duplication"`
	Classification string `json:"classification"`
	Justification  string `json:"justification"`
	IntegrityHash  string `json:"integrity_hash,omitempty"`
}

type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(filesystem fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: filesystem, printer: printer}
}

func (s *Service) Generate(root string) ([]Entry, error) {
	entries, err := s.Inventory(root)
	if err != nil {
		return nil, err
	}
	markdown, err := s.markdown(entries)
	if err != nil {
		return nil, err
	}
	data, err := s.json(entries)
	if err != nil {
		return nil, err
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(root, "docs", "hook-inventory.md"), markdown); err != nil {
		return nil, fmt.Errorf("write hook inventory markdown: %w", err)
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(root, "testdata", "hook-inventory.json"), data); err != nil {
		return nil, fmt.Errorf("write hook inventory json: %w", err)
	}
	s.printer.Info("Inventario de hooks gerado: %d entradas", len(entries))
	return entries, nil
}

func (s *Service) Check(root string) error {
	entries, err := s.Inventory(root)
	if err != nil {
		return err
	}
	markdown, err := s.markdown(entries)
	if err != nil {
		return err
	}
	data, err := s.json(entries)
	if err != nil {
		return err
	}
	if err := s.matches(filepath.Join(root, "docs", "hook-inventory.md"), markdown); err != nil {
		return err
	}
	if err := s.matches(filepath.Join(root, "testdata", "hook-inventory.json"), data); err != nil {
		return err
	}
	return nil
}

func (s *Service) Inventory(root string) ([]Entry, error) {
	locations := []string{
		".agents/hooks", ".claude/hooks", ".codex/hooks", ".github/hooks", ".agents/scripts",
		".claude/scripts", "internal/runtime/hooks", ".opencode/plugin", ".agents/lib", "scripts/git-hooks",
	}
	entries := make([]Entry, 0)
	for _, location := range locations {
		found, err := s.scan(root, location)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Location < entries[j].Location })
	return entries, nil
}

func (s *Service) scan(root, location string) ([]Entry, error) {
	entries, err := s.fs.ReadDir(filepath.Join(root, location))
	if err != nil {
		return nil, fmt.Errorf("read hook source %s: %w", location, err)
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == "dispatcher.go" {
			continue
		}
		path := filepath.Join(location, entry.Name())
		if location == "internal/runtime/hooks" && filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		data, err := s.fs.ReadFile(filepath.Join(root, path))
		if err != nil {
			return nil, fmt.Errorf("read hook %s: %w", path, err)
		}
		items = append(items, s.newEntry(filepath.ToSlash(path), data))
	}
	return items, nil
}

func (s *Service) newEntry(location string, data []byte) Entry {
	entry := Entry{
		Name:           filepath.Base(location),
		Location:       location,
		Event:          "lifecycle",
		Provider:       s.provider(location),
		Purpose:        "Aplica governanca operacional no ponto de hook declarado.",
		PolicyGate:     "Politicas em .agents/policies/ nao sao lidas diretamente; gate auto-contido.",
		Blocking:       "NAO BLOQUEANTE",
		Cost:           "baixo",
		FailureMode:    "FAIL-CLOSED",
		TestCoverage:   "COM TESTE",
		Duplication:    s.duplication(location),
		Classification: "KEEP",
		Justification:  "Mantido por proteger governanca ativa ou por ser espelho distribuido.",
	}
	if location == ".agents/scripts/validate-governance-references.sh" {
		entry.TestCoverage = "SEM TESTE"
		entry.Classification = "ON-DEMAND"
		entry.Justification = "Orfao de invocacao: nao possui call-site executavel no repositorio."
	}
	if strings.Contains(location, ".claude/hooks/post-wave.sh") {
		entry.TestCoverage = "SEM TESTE"
	}
	if strings.Contains(location, "scripts/git-hooks/pre-commit") {
		entry.TestCoverage = "COM TESTE (parcial: apenas bloco 3; scripts/test-hooks.sh:538)"
		entry.FailureMode = "FAIL-OPEN (permissivo; linhas 41,54,58,61,66)"
	}
	if strings.Contains(location, "validate-token-budget.sh") {
		entry.Classification = "ON-DEMAND"
		entry.Justification = "LOCAL-ONLY inerte: allowlist CLAUDE_LOCAL_ONLY_HOOKS impede espelhamento e nao ha registro em settings."
	}
	if failureMode, ok := s.failOpen(location); ok {
		entry.FailureMode = failureMode
	}
	if s.critical(location) {
		hash := sha256.Sum256(data)
		entry.IntegrityHash = hex.EncodeToString(hash[:])
	}
	return entry
}

func (s *Service) provider(location string) string {
	switch {
	case strings.HasPrefix(location, ".claude/"):
		return "Claude"
	case strings.HasPrefix(location, ".codex/"):
		return "Codex"
	case strings.HasPrefix(location, ".github/"):
		return "Copilot"
	case strings.HasPrefix(location, ".opencode/"):
		return "OpenCode"
	case strings.HasPrefix(location, "internal/runtime/"):
		return "Runtime ACP"
	default:
		return "Vendor-neutral"
	}
}

func (s *Service) duplication(location string) string {
	if strings.HasPrefix(location, ".agents/") || strings.HasPrefix(location, ".claude/") {
		return "Possui espelhos por provedor ou distribuicao embarcada."
	}
	return "Sem espelho equivalente inventariado."
}

func (s *Service) failOpen(location string) (string, bool) {
	modes := map[string]string{
		".agents/hooks/subagent-stop-wrapper.sh": "FAIL-OPEN (.agents/hooks/subagent-stop-wrapper.sh:30,35,40,87,101)",
		".agents/hooks/validate-governance.sh":   "FAIL-OPEN (.agents/hooks/validate-governance.sh:33,38)",
		".agents/scripts/hook-prereq-gate.sh":    "FAIL-OPEN (.agents/scripts/hook-prereq-gate.sh:50,60)",
		".agents/hooks/validate-preload.sh":      "FAIL-OPEN (.agents/hooks/validate-preload.sh:54,65)",
		".agents/scripts/git-operation-gate.sh":  "FAIL-OPEN (.agents/scripts/git-operation-gate.sh:39-41)",
	}
	mode, ok := modes[location]
	return mode, ok
}

func (s *Service) critical(location string) bool {
	return strings.Contains(location, "validate-") || strings.Contains(location, "gate") || strings.Contains(location, "pre-commit")
}

func (s *Service) markdown(entries []Entry) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("# Inventario de Hooks\n\n")
	builder.WriteString("Gerado por `ai-spec hooks inventory`; nao editar manualmente.\n\n")
	builder.WriteString("| Nome | Localizacao | Evento | Provedor | Objetivo | Policy/gate | Bloqueante | Custo | Falha esperada | Cobertura | Duplicacao | Classificacao | Justificativa | Integridade |\n")
	builder.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, entry := range entries {
		values := []string{entry.Name, entry.Location, entry.Event, entry.Provider, entry.Purpose, entry.PolicyGate, entry.Blocking, entry.Cost, entry.FailureMode, entry.TestCoverage, entry.Duplication, entry.Classification, entry.Justification, entry.IntegrityHash}
		for index, value := range values {
			if index > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(strings.ReplaceAll(value, "|", "/"))
		}
		builder.WriteString("\n")
	}
	return []byte(builder.String()), nil
}

func (s *Service) json(entries []Entry) ([]byte, error) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hook inventory: %w", err)
	}
	return append(data, '\n'), nil
}

func (s *Service) matches(path string, expected []byte) error {
	actual, err := s.fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated inventory %s: %w", path, err)
	}
	if string(actual) != string(expected) {
		return fmt.Errorf("hook inventory drift: %s", path)
	}
	return nil
}
