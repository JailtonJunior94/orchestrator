package uninstall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

type remover struct {
	fs      fs.FileSystem
	printer *output.Printer
	dryRun  bool
	count   int
}

func (r *remover) remove(target string) {
	if !r.fs.Exists(target) && !r.fs.IsDir(target) {
		return
	}
	if r.dryRun {
		r.printer.DryRun("rm %s", target)
	} else {
		_ = r.fs.RemoveAll(target)
	}
	r.count++
}

func (r *remover) removeDir(target, reason string) {
	if !r.fs.IsDir(target) {
		return
	}
	if r.dryRun {
		r.printer.DryRun("rm -r %s (%s)", target, reason)
	} else {
		_ = r.fs.RemoveAll(target)
	}
	r.count++
}

func (r *remover) rewrite(target string, data []byte) {
	if r.dryRun {
		r.printer.DryRun("reescrever %s sem o bloco de governanca", target)
		r.count++
		return
	}
	if err := r.fs.WriteFile(target, data); err != nil {
		return
	}
	r.count++
}

var claudeGovernanceHookCommands = map[string]bool{
	"bash .claude/hooks/validate-preload.sh":      true,
	"bash .claude/hooks/validate-governance.sh":   true,
	"bash .claude/hooks/validate-session-end.sh":  true,
	"bash .claude/hooks/subagent-stop-wrapper.sh": true,
	"bash .claude/hooks/post-execute-task.sh":     true,
	"bash .claude/hooks/pre-execute-all-tasks.sh": true,
	"bash .claude/hooks/post-wave.sh":             true,
}

const (
	codexConfigRelPath     = ".codex/config.toml"
	agentsGeneratedMarker  = "<!-- governance-schema:"
	codexSkillsConfigTable = "[[skills.config]]"
	codexSkillsPathPrefix  = `path = ".agents/skills/`
)

var toolMarkdownGeneratedMarkers = []string{
	"Use `AGENTS.md` como",
	"## Instrucoes",
	"1. Ler `AGENTS.md` no inicio da sessao.",
}

var governanceRootDirs = []string{
	".agents",
	".claude",
	".codex",
	".github",
	".opencode",
	"scripts",
}

var claudeSettingsRelPath = filepath.Join(".claude", "settings.local.json")

var reverseMergeFiles = []string{
	filepath.Join(".github", "settings.json"),
	specs.OpenCodeConfigFileName,
}

func (s *Service) Execute(projectDir string, dryRun bool) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if !s.fs.IsDir(absDir) {
		return fmt.Errorf("diretorio alvo nao encontrado: %s", absDir)
	}

	s.printer.Info("Removendo governanca de: %s", absDir)
	s.printer.Info("")

	rm := &remover{fs: s.fs, printer: s.printer, dryRun: dryRun}

	mf, loadErr := manifest.NewStore(s.fs).Load(absDir)
	tracked := loadErr == nil && mf.HasFileTracking()
	switch {
	case tracked:
		for _, rel := range mf.InstalledFiles {
			rm.remove(filepath.Join(absDir, rel))
		}
	case loadErr == nil:
		s.printer.Warn("manifesto sem rastreamento por arquivo (versao anterior a esta funcionalidade) — removendo apenas o conjunto conservador conhecido; reinstale para habilitar remocao completa por arquivo.")
		s.legacyStaticRemoval(absDir, rm)
	default:
		s.printer.Warn("manifesto sem rastreamento por arquivo: %s ausente — removendo apenas arquivos cujo conteudo e reconhecido como gerado pelo harness; arquivos do usuario sao preservados.", manifest.ManifestFile)
		s.removeGeneratedContextFiles(absDir, rm)
	}

	for _, rel := range s.mergedCandidates(mf, tracked) {
		s.reverseMerge(absDir, rel, rm)
	}

	s.removeClaudeSettings(absDir, rm)
	s.removeLegacyGeminiResidue(absDir, rm)

	rm.remove(filepath.Join(absDir, manifest.ManifestFile))

	s.pruneEmptyDirs(absDir, dryRun)

	localFile := filepath.Join(absDir, "AGENTS.local.md")
	if s.fs.Exists(localFile) {
		s.printer.Info("")
		s.printer.Info("AGENTS.local.md preservado (extensao local do usuario).")
	}

	s.printer.Info("")
	switch {
	case rm.count == 0:
		s.printer.Info("Nada a remover: nenhum artefato de governanca encontrado em %s.", absDir)
	case dryRun:
		s.printer.Info("[dry-run] %d arquivo(s) seriam removidos. Nenhuma alteracao feita.", rm.count)
	default:
		s.printer.Info("Governanca removida: %d arquivo(s).", rm.count)
	}

	return nil
}

func (s *Service) mergedCandidates(mf *manifest.Manifest, tracked bool) []string {
	seen := make(map[string]bool, len(reverseMergeFiles))
	out := make([]string, 0, len(reverseMergeFiles))
	for _, rel := range reverseMergeFiles {
		seen[rel] = true
		out = append(out, rel)
	}
	if !tracked {
		return out
	}
	for _, rel := range mf.MergedFiles {
		clean := filepath.Clean(rel)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

func (s *Service) reverseMerge(absDir, rel string, rm *remover) {
	path := filepath.Join(absDir, rel)
	switch filepath.ToSlash(rel) {
	case ".github/settings.json":
		s.reverseMergeCopilotSettings(path, rm)
	case specs.OpenCodeConfigFileName:
		s.reverseMergeOpenCodeConfig(path, rm)
	case codexConfigRelPath:
		s.reverseMergeCodexConfig(path, rm)
	case claudeSettingsRelPath:
		s.reverseMergeClaudeSettings(path, rm)
	default:
		if s.reverseMergeUserContentMarkdown(path, rm) {
			return
		}
		s.printer.Warn("%s foi mesclado na instalacao mas nao tem merge reverso conhecido — mantido como esta; revise manualmente o conteudo de governanca remanescente.", filepath.ToSlash(rel))
	}
}

func (s *Service) reverseMergeUserContentMarkdown(path string, rm *remover) bool {
	if !s.fs.Exists(path) {
		return false
	}
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return false
	}
	preserved, ok := contextgen.ExtractUserContentBlock(string(data))
	if !ok {
		return false
	}
	if strings.TrimSpace(preserved) == "" {
		rm.remove(path)
		return true
	}
	rm.rewrite(path, []byte(preserved))
	return true
}

func (s *Service) reverseMergeCodexConfig(path string, rm *remover) {
	if !s.fs.Exists(path) {
		return
	}
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	preserved := contextgen.StripCodexGeneratedRegion(content)
	if preserved == content {
		return
	}
	if strings.TrimSpace(preserved) == "" {
		rm.remove(path)
		return
	}
	rm.rewrite(path, []byte(preserved))
}

func (s *Service) reverseMergeCopilotSettings(path string, rm *remover) {
	raw, rawOK := s.readRawFile(path)
	doc, ok := s.readJSONObject(path)
	if !ok {
		return
	}
	hooks, hasHooks := doc["hooks"].(map[string]any)
	if !hasHooks {
		return
	}
	removed := false
	for event, rawEntries := range hooks {
		entries, isList := rawEntries.([]any)
		if !isList {
			continue
		}
		kept := make([]any, 0, len(entries))
		for _, entry := range entries {
			if s.isCopilotGovernanceHook(entry) {
				removed = true
				continue
			}
			kept = append(kept, entry)
		}
		if len(kept) == len(entries) {
			continue
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = kept
	}
	if !removed {
		return
	}
	if len(hooks) == 0 {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooks
	}
	if rawOK && s.persistPreservingLayout(path, raw, doc, "hooks", hooks, rm) {
		return
	}
	s.persistOrRemove(path, doc, rm)
}

func (s *Service) reverseMergeClaudeSettings(path string, rm *remover) {
	raw, rawOK := s.readRawFile(path)
	doc, ok := s.readJSONObject(path)
	if !ok {
		return
	}
	hooks, hasHooks := doc["hooks"].(map[string]any)
	if !hasHooks {
		return
	}
	removed := false
	for event, rawEntries := range hooks {
		entries, isList := rawEntries.([]any)
		if !isList {
			continue
		}
		kept := make([]any, 0, len(entries))
		changedEvent := false
		for _, entry := range entries {
			stripped, changed := s.stripClaudeGovernanceHooks(entry)
			if changed {
				changedEvent = true
				removed = true
			}
			if stripped == nil {
				continue
			}
			kept = append(kept, stripped)
		}
		if !changedEvent {
			continue
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = kept
	}
	if !removed {
		return
	}
	if len(hooks) == 0 {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooks
	}
	if rawOK && s.persistPreservingLayout(path, raw, doc, "hooks", hooks, rm) {
		return
	}
	s.persistOrRemove(path, doc, rm)
}

func (s *Service) stripClaudeGovernanceHooks(entry any) (any, bool) {
	matcher, ok := entry.(map[string]any)
	if !ok {
		return entry, false
	}
	nested, isList := matcher["hooks"].([]any)
	if !isList {
		return entry, false
	}
	kept := make([]any, 0, len(nested))
	for _, hook := range nested {
		command, _ := hook.(map[string]any)
		if text, isText := command["command"].(string); isText && strings.Contains(text, ".claude/hooks/") {
			continue
		}
		kept = append(kept, hook)
	}
	if len(kept) == len(nested) {
		return entry, false
	}
	if len(kept) == 0 {
		return nil, true
	}
	matcher["hooks"] = kept
	return matcher, true
}

func (s *Service) isCopilotGovernanceHook(entry any) bool {
	hook, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	command, _ := hook["bash"].(string)
	return strings.Contains(command, ".github/hooks/")
}

func (s *Service) reverseMergeOpenCodeConfig(path string, rm *remover) {
	raw, rawOK := s.readRawFile(path)
	doc, ok := s.readJSONObject(path)
	if !ok {
		return
	}
	permission, hasPermission := doc["permission"].(map[string]any)
	if !hasPermission {
		return
	}
	for tool, rule := range specs.DefaultOpenCodePermission() {
		installedRule, ruleIsMap := rule.(map[string]any)
		currentRule, currentIsMap := permission[tool].(map[string]any)
		if ruleIsMap && currentIsMap {
			for command, decision := range installedRule {
				if current, found := currentRule[command]; found && fmt.Sprint(current) == fmt.Sprint(decision) {
					delete(currentRule, command)
				}
			}
			if len(currentRule) == 0 {
				delete(permission, tool)
				continue
			}
			permission[tool] = currentRule
			continue
		}
		if current, found := permission[tool]; found && fmt.Sprint(current) == fmt.Sprint(rule) {
			delete(permission, tool)
		}
	}
	if len(permission) == 0 {
		delete(doc, "permission")
	} else {
		doc["permission"] = permission
	}
	if rawOK && s.persistPreservingLayout(path, raw, doc, "permission", permission, rm) {
		return
	}
	s.persistOrRemove(path, doc, rm)
}

func (s *Service) persistPreservingLayout(
	path string,
	raw []byte,
	doc map[string]any,
	key string,
	value map[string]any,
	rm *remover,
) bool {
	if len(doc) == 0 {
		rm.remove(path)
		return true
	}

	catalog := specs.NewCatalog()
	var updated []byte
	var err error
	if len(value) == 0 {
		updated, err = catalog.DeleteJSONTopLevelKey(raw, key)
	} else {
		updated, err = catalog.SetJSONTopLevelKey(raw, key, value)
	}
	if err != nil {
		return false
	}
	if bytes.Equal(raw, updated) {
		return true
	}
	rm.rewrite(path, updated)
	return true
}

func (s *Service) readRawFile(path string) ([]byte, bool) {
	if !s.fs.Exists(path) {
		return nil, false
	}
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func (s *Service) readJSONObject(path string) (map[string]any, bool) {
	if !s.fs.Exists(path) {
		return nil, false
	}
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, false
	}
	doc := map[string]any{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false
	}
	return doc, true
}

func (s *Service) persistOrRemove(path string, doc map[string]any, rm *remover) {
	if len(doc) == 0 {
		rm.remove(path)
		return
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	encoded = append(encoded, '\n')
	if current, readErr := s.fs.ReadFile(path); readErr == nil && bytes.Equal(current, encoded) {
		return
	}
	rm.rewrite(path, encoded)
}

func (s *Service) removeGeneratedDir(rm *remover, dir, suffix string) {
	if !s.fs.IsDir(dir) {
		return
	}
	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if suffix != "" && !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		rm.remove(filepath.Join(dir, entry.Name()))
	}
}

func (s *Service) removeClaudeSettings(absDir string, rm *remover) {
	settingsFile := filepath.Join(absDir, ".claude", "settings.local.json")
	if !s.fs.Exists(settingsFile) {
		return
	}
	data, err := s.fs.ReadFile(settingsFile)
	if err != nil {
		return
	}
	content := string(data)
	if s.isGeneratedClaudeSettings(content) {
		rm.remove(settingsFile)
		return
	}
	if strings.Contains(content, "validate-governance") || strings.Contains(content, "validate-preload") {
		s.printer.Warn(".claude/settings.local.json contem configuracoes alem dos hooks de governanca — mantido.")
	}
}

func (s *Service) removeLegacyGeminiResidue(absDir string, rm *remover) {
	s.removeIfGenerated(absDir, "GEMINI.md", s.isGeneratedToolMarkdown, rm)
	rm.removeDir(filepath.Join(absDir, ".gemini"), "residuo legado do agente removido")
}

func (s *Service) pruneEmptyDirs(absDir string, dryRun bool) {
	if dryRun {
		return
	}
	for _, rel := range governanceRootDirs {
		s.pruneEmptyTree(filepath.Join(absDir, rel))
	}
}

func (s *Service) pruneEmptyTree(dir string) bool {
	if !s.fs.IsDir(dir) {
		return false
	}
	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		return false
	}
	remaining := 0
	for _, entry := range entries {
		if entry.IsDir() && s.pruneEmptyTree(filepath.Join(dir, entry.Name())) {
			continue
		}
		remaining++
	}
	if remaining > 0 {
		return false
	}
	return s.fs.RemoveAll(dir) == nil
}

func (s *Service) legacyStaticRemoval(absDir string, rm *remover) {
	s.legacyGeneratedSweep(absDir, rm)
	for _, rel := range []string{
		filepath.Join(".claude", "rules", "governance.md"),
		filepath.Join(".claude", "scripts", "validate-task-evidence.sh"),
		filepath.Join(".claude", "scripts", "validate-bugfix-evidence.sh"),
		filepath.Join(".claude", "scripts", "validate-refactor-evidence.sh"),
		filepath.Join(".claude", "hooks", "validate-governance.sh"),
		filepath.Join(".claude", "hooks", "validate-preload.sh"),
		filepath.Join("scripts", "lib", "parse-hook-input.sh"),
		filepath.Join("scripts", "lib", "check-invocation-depth.sh"),
	} {
		rm.remove(filepath.Join(absDir, rel))
	}
	s.removeGeneratedContextFiles(absDir, rm)
}

func (s *Service) legacyGeneratedSweep(absDir string, rm *remover) {
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".agents", "skills"), "")
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".claude", "skills"), "")
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".claude", "agents"), ".md")
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".github", "skills"), "")
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".github", "agents"), ".agent.md")
	s.removeGeneratedDir(rm, filepath.Join(absDir, ".codex", "agents"), ".toml")
}

func (s *Service) removeGeneratedContextFiles(absDir string, rm *remover) {
	s.removeIfGenerated(absDir, "AGENTS.md", s.isGeneratedAgentsMarkdown, rm)
	s.removeIfGenerated(absDir, "CLAUDE.md", s.isGeneratedToolMarkdown, rm)
	s.removeIfGenerated(absDir, filepath.Join(".github", "copilot-instructions.md"), s.isGeneratedToolMarkdown, rm)
	s.removeIfGenerated(absDir, filepath.Join(".codex", "config.toml"), s.isGeneratedCodexConfig, rm)
}

func (s *Service) removeIfGenerated(absDir, rel string, recognize func(string) bool, rm *remover) {
	path := filepath.Join(absDir, rel)
	if !s.fs.Exists(path) {
		return
	}
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return
	}
	if s.reverseMergeUserContentMarkdown(path, rm) {
		return
	}
	if contextgen.StripCodexGeneratedRegion(string(data)) != string(data) {
		s.reverseMergeCodexConfig(path, rm)
		return
	}
	if recognize(string(data)) {
		rm.remove(path)
		return
	}
	s.printer.Warn("%s nao corresponde ao conteudo gerado pelo harness — preservado; remova manualmente se nao precisar mais dele.", filepath.ToSlash(rel))
}

func (s *Service) isGeneratedAgentsMarkdown(content string) bool {
	return strings.HasPrefix(strings.TrimLeft(content, " \t\r\n"), agentsGeneratedMarker)
}

func (s *Service) isGeneratedToolMarkdown(content string) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "# ") {
		return false
	}
	for _, marker := range toolMarkdownGeneratedMarkers {
		if !strings.Contains(trimmed, marker) {
			return false
		}
	}
	return true
}

func (s *Service) isGeneratedCodexConfig(content string) bool {
	tables := 0
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
		case line == codexSkillsConfigTable:
			tables++
		case line == "enabled = true":
		case strings.HasPrefix(line, codexSkillsPathPrefix):
		default:
			return false
		}
	}
	return tables > 0
}

func (s *Service) isGeneratedClaudeSettings(content string) bool {
	doc := map[string]any{}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return false
	}
	if len(doc) != 1 {
		return false
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok || len(hooks) == 0 {
		return false
	}
	for _, raw := range hooks {
		matchers, isList := raw.([]any)
		if !isList {
			return false
		}
		for _, rawMatcher := range matchers {
			matcher, isObject := rawMatcher.(map[string]any)
			if !isObject {
				return false
			}
			commands, hasCommands := matcher["hooks"].([]any)
			if !hasCommands {
				return false
			}
			for _, rawCommand := range commands {
				hook, isHook := rawCommand.(map[string]any)
				if !isHook {
					return false
				}
				command, _ := hook["command"].(string)
				if !claudeGovernanceHookCommands[strings.TrimSpace(command)] {
					return false
				}
			}
		}
	}
	return true
}
