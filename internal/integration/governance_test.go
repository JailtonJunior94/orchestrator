//go:build integration

package integration

// Harness de regressao de governanca — TASK-16
//
// Verifica contratos criticos que devem ser satisfeitos automaticamente para
// garantir que alteracoes em SKILL.md, references ou hooks nao quebrem
// silenciosamente o fluxo de carregamento ou a paridade entre agentes.
//
// Contratos verificados:
//  1. Toda skill embarcada tem SKILL.md com frontmatter valido (JSON Schema).
//  2. Toda referencia citada em SKILL.md existe no caminho esperado.
//  3. Hooks referenciados em settings.local.json existem e sao executaveis.
//  4. skills-lock.json e consistente com o diretorio .agents/skills/.
//  5. Paridade semantica entre artefatos Claude / Gemini / Codex passa os invariantes.

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/parity"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// govRepoRoot retorna o caminho absoluto da raiz do repositorio.
// O arquivo esta em internal/integration/, entao sobe dois niveis.
func govRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("resolver raiz do repo: %v", err)
	}
	return dir
}

// govEmbeddedSkillsDir retorna o diretorio de skills embarcadas.
func govEmbeddedSkillsDir(root string) string {
	return filepath.Join(root, "internal", "embedded", "assets", ".agents", "skills")
}

// referencesRegexp extrai mencoes a references/xxx.md de dentro de backticks.
// Captura apenas referencias relativas a propria skill (sem prefixo de outra skill).
var referencesRegexp = regexp.MustCompile("`references/([a-zA-Z0-9._-]+\\.md)`")

// ── Subtask 16.2 — Contrato 1: frontmatter valido via JSON Schema ─────────────

// TestGov16_EmbeddedSkills_ValidFrontmatterSchema verifica que todas as skills
// embarcadas possuem SKILL.md com frontmatter valido segundo o JSON Schema formal.
// Este teste e redundante com internal/skills/contract_test.go mas garante que
// make integration execute a verificacao no mesmo pipeline.
func TestGov16_EmbeddedSkills_ValidFrontmatterSchema(t *testing.T) {
	root := govRepoRoot(t)
	embeddedDir := govEmbeddedSkillsDir(root)

	entries, err := os.ReadDir(embeddedDir)
	if err != nil {
		t.Fatalf("ler diretorio de skills embarcadas: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		t.Run(skillName, func(t *testing.T) {
			t.Parallel()
			skillMD := filepath.Join(embeddedDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			if err != nil {
				t.Fatalf("SKILL.md nao encontrado em %q: %v", skillName, err)
			}
			if err := skills.ValidateFrontmatterSchema(data, skillName); err != nil {
				t.Errorf("skill embarcada %q falhou no JSON Schema: %v", skillName, err)
			}
		})
	}
}

// ── Subtask 16.2 — Contrato 2: referencias existem no filesystem ──────────────

// TestGov16_EmbeddedSkills_ReferencesExist verifica que toda referencia citada em
// um SKILL.md embarcado existe como arquivo no diretorio references/ da skill.
// Previne quebra silenciosa quando um references/ e renomeado ou removido.
func TestGov16_EmbeddedSkills_ReferencesExist(t *testing.T) {
	root := govRepoRoot(t)
	embeddedDir := govEmbeddedSkillsDir(root)

	entries, err := os.ReadDir(embeddedDir)
	if err != nil {
		t.Fatalf("ler diretorio de skills embarcadas: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		t.Run(skillName, func(t *testing.T) {
			t.Parallel()
			skillMD := filepath.Join(embeddedDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			if err != nil {
				t.Fatalf("SKILL.md nao encontrado em %q: %v", skillName, err)
			}

			matches := referencesRegexp.FindAllSubmatch(data, -1)
			for _, m := range matches {
				refFile := string(m[1])
				refPath := filepath.Join(embeddedDir, skillName, "references", refFile)
				if _, err := os.Stat(refPath); err != nil {
					t.Errorf("skill %q referencia %q mas o arquivo nao existe: %s",
						skillName, "references/"+refFile, refPath)
				}
			}
		})
	}
}

// ── Subtask 16.2 — Contrato 3: hooks existem e sao executaveis ────────────────

// settingsHook representa um hook individual no settings.local.json.
type settingsHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// settingsHookGroup agrupa hooks por matcher.
type settingsHookGroup struct {
	Matcher string         `json:"matcher"`
	Hooks   []settingsHook `json:"hooks"`
}

// settingsFile representa a estrutura raiz de settings.local.json.
type settingsFile struct {
	Hooks map[string][]settingsHookGroup `json:"hooks"`
}

// TestGov16_Hooks_ExistAndExecutable verifica que os hooks registrados em
// .claude/settings.local.json existem no filesystem e possuem permissao de execucao.
// Um hook inexistente ou sem permissao 0o111 bloqueia silenciosamente o fluxo de Claude.
func TestGov16_Hooks_ExistAndExecutable(t *testing.T) {
	root := govRepoRoot(t)
	settingsPath := filepath.Join(root, ".claude", "settings.local.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Skipf("settings.local.json nao encontrado (skip em repos sem Claude instalado): %v", err)
	}

	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("parsear settings.local.json: %v", err)
	}

	// Extrair todos os caminhos de script dos campos command.
	// Formato esperado: "bash <path>" ou "<path>" sozinho.
	seen := make(map[string]bool)
	for phase, groups := range sf.Hooks {
		for _, group := range groups {
			for _, hook := range group.Hooks {
				scriptPath := extractScriptPath(hook.Command)
				if scriptPath == "" || seen[scriptPath] {
					continue
				}
				seen[scriptPath] = true

				absPath := filepath.Join(root, scriptPath)
				t.Run(phase+"/"+scriptPath, func(t *testing.T) {
					info, err := os.Stat(absPath)
					if err != nil {
						t.Errorf("hook %q referenciado em %s nao existe: %v", scriptPath, phase, err)
						return
					}
					if info.Mode()&0o111 == 0 {
						t.Errorf("hook %q referenciado em %s nao tem permissao de execucao (mode=%s)",
							scriptPath, phase, info.Mode())
					}
				})
			}
		}
	}

	if len(seen) == 0 {
		t.Log("nenhum hook com script encontrado em settings.local.json")
	}
}

// extractScriptPath extrai o caminho do script de um campo command de hook.
// Suporta: "bash <path>", "sh <path>", "<path>" direto.
func extractScriptPath(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	parts := strings.Fields(command)
	switch {
	case len(parts) >= 2 && (parts[0] == "bash" || parts[0] == "sh"):
		return parts[1]
	case len(parts) == 1 && strings.HasSuffix(parts[0], ".sh"):
		return parts[0]
	}
	return ""
}

// ── Subtask 16.2 — Contrato 4: skills-lock.json consistente com .agents/skills/ ──

// lockFile representa a estrutura de skills-lock.json.
type lockFile struct {
	Version int                     `json:"version"`
	Skills  map[string]lockEntry    `json:"skills"`
}

type lockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	ComputedHash string `json:"computedHash"`
}

// TestGov16_SkillsLock_LockedSkillsHaveDirectory verifica que cada skill registrada
// em skills-lock.json tem um diretorio correspondente em .agents/skills/.
// Um registro sem diretorio indica que a skill foi removida sem atualizar o lock.
func TestGov16_SkillsLock_LockedSkillsHaveDirectory(t *testing.T) {
	root := govRepoRoot(t)
	lockPath := filepath.Join(root, "skills-lock.json")

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ler skills-lock.json: %v", err)
	}
	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("parsear skills-lock.json: %v", err)
	}

	installedDir := filepath.Join(root, ".agents", "skills")

	for skillName := range lock.Skills {
		skillName := skillName
		t.Run(skillName, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(installedDir, skillName)
			info, err := os.Stat(dir)
			if err != nil {
				t.Errorf("skill %q esta no skills-lock.json mas o diretorio nao existe: %s", skillName, dir)
				return
			}
			if !info.IsDir() {
				t.Errorf("esperado diretorio para skill %q, encontrado arquivo: %s", skillName, dir)
			}
		})
	}
}

// TestGov16_SkillsLock_ExternalSkillsHaveLockEntry verifica que cada diretorio
// em .agents/skills/ que nao e uma skill base ou de linguagem esta no skills-lock.json.
// Previne que skills externas instaladas manualmente fiquem sem rastreamento de hash.
func TestGov16_SkillsLock_ExternalSkillsHaveLockEntry(t *testing.T) {
	root := govRepoRoot(t)
	lockPath := filepath.Join(root, "skills-lock.json")

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ler skills-lock.json: %v", err)
	}
	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("parsear skills-lock.json: %v", err)
	}

	// Construir conjunto de skills gerenciadas pelo harness (nao precisam de lock entry)
	managed := make(map[string]bool)
	for _, s := range skills.BaseSkills {
		managed[s] = true
	}
	for _, s := range skills.LangSkills(skills.AllLangs) {
		managed[s] = true
	}

	installedDir := filepath.Join(root, ".agents", "skills")
	entries, err := os.ReadDir(installedDir)
	if err != nil {
		t.Fatalf("ler diretorio .agents/skills/: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()

		// Ignorar diretorios sem SKILL.md (ex: diretorio tests/ de Python)
		if _, err := os.Stat(filepath.Join(installedDir, skillName, "SKILL.md")); err != nil {
			continue
		}

		// Skills gerenciadas pelo harness nao precisam de entrada no lock
		if managed[skillName] {
			continue
		}

		t.Run(skillName, func(t *testing.T) {
			if _, ok := lock.Skills[skillName]; !ok {
				t.Errorf("skill %q esta em .agents/skills/ mas nao tem entrada no skills-lock.json", skillName)
			}
		})
	}
}

// ── Subtask 16.3 — Paridade entre agentes ─────────────────────────────────────

// TestGov16_Parity_InvariantsPass_AllTools executa o conjunto completo de invariantes
// de paridade para a combinacao Claude + Gemini + Codex + Copilot.
// Falhas Common ou ToolSpecific bloqueiam; BestEffort sao registradas como aviso.
func TestGov16_Parity_InvariantsPass_AllTools(t *testing.T) {
	runGovParity(t, skills.AllTools, "full")
}

// TestGov16_Parity_InvariantsPass_ClaudeOnly executa invariantes para Claude isolado.
// Verifica que hooks e rules Claude estao presentes quando apenas Claude e instalado.
func TestGov16_Parity_InvariantsPass_ClaudeOnly(t *testing.T) {
	runGovParity(t, []skills.Tool{skills.ToolClaude}, "full")
}

// TestGov16_Parity_InvariantsPass_ClaudeGemini executa invariantes para Claude + Gemini.
// Verifica paridade de caminho canonico e documentacao de best-effort no Gemini.
func TestGov16_Parity_InvariantsPass_ClaudeGemini(t *testing.T) {
	runGovParity(t, []skills.Tool{skills.ToolClaude, skills.ToolGemini}, "full")
}

// TestGov16_Parity_InvariantsPass_CodexOnly executa invariantes para Codex isolado.
// Verifica que profile compact e aplicado (sem secoes verbose no AGENTS.md).
func TestGov16_Parity_InvariantsPass_CodexOnly(t *testing.T) {
	runGovParity(t, []skills.Tool{skills.ToolCodex}, "compact")
}

// runGovParity e o helper que executa os invariantes de paridade e reporta resultados.
// Invariantes Common e ToolSpecific causam t.Errorf; BestEffort causam t.Logf.
func runGovParity(t *testing.T, tools []skills.Tool, profile string) {
	t.Helper()

	snap, err := parity.Generate("/project-gov-test", tools, nil, profile)
	if err != nil {
		t.Fatalf("gerar snapshot de paridade: %v", err)
	}

	results := parity.Run(snap, parity.Invariants())
	for _, cr := range results {
		if cr.Skipped || cr.Result.OK {
			continue
		}
		switch cr.Invariant.Level {
		case parity.Common, parity.ToolSpecific:
			t.Errorf("[%s] FALHOU (%s): %s — %s",
				cr.Invariant.ID, cr.Invariant.Level,
				cr.Invariant.Description, cr.Result.Reason)
		case parity.BestEffort:
			t.Logf("[%s] AVISO (best-effort): %s — %s",
				cr.Invariant.ID,
				cr.Invariant.Description, cr.Result.Reason)
		}
	}
}

// TestGov16_Parity_InvariantDivergence_ReportsSkillAndAgent verifica que ao remover
// um artefato esperado de uma ferramenta, o invariante correspondente falha com
// mensagem que identifica a ferramenta e o artefato ausente.
// Este teste documenta o comportamento de diagnostico do harness.
func TestGov16_Parity_InvariantDivergence_ReportsSkillAndAgent(t *testing.T) {
	snap, err := parity.Generate("/project-gov-divergence", []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("gerar snapshot: %v", err)
	}

	// Remover o hook de governanca para simular divergencia
	hookPath := "/project-gov-divergence/.claude/hooks/validate-governance.sh"
	delete(snap.Files, hookPath)

	results := parity.Run(snap, parity.Invariants())

	foundFailure := false
	for _, cr := range results {
		if cr.Skipped || cr.Result.OK {
			continue
		}
		if cr.Invariant.ID == "CL03" {
			foundFailure = true
			if !strings.Contains(cr.Result.Reason, "validate-governance.sh") {
				t.Errorf("mensagem de falha do invariante CL03 deveria mencionar o artefato ausente, got: %q",
					cr.Result.Reason)
			}
		}
	}

	if !foundFailure {
		t.Error("esperado falha no invariante CL03 ao remover validate-governance.sh, mas nenhuma falha encontrada")
	}
}

// Garante que o package fs importado nao conflite com o nome local.
var _ fs.FileMode = 0
