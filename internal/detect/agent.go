package detect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// LookPather abstrai exec.LookPath para injecao em testes.
type LookPather interface {
	LookPath(file string) (string, error)
}

// OSLookPather implementa LookPather usando exec.LookPath real.
type OSLookPather struct{}

// LookPath delega para exec.LookPath do sistema.
func (OSLookPather) LookPath(file string) (string, error) { return exec.LookPath(file) }

// HomeDir abstrai os.UserHomeDir para injecao em testes (R-SEC-001).
type HomeDir interface {
	UserHomeDir() (string, error)
}

// OSHomeDir implementa HomeDir usando os.UserHomeDir real.
type OSHomeDir struct{}

// UserHomeDir delega para os.UserHomeDir.
func (OSHomeDir) UserHomeDir() (string, error) { return os.UserHomeDir() }

// DetectOptions parametriza a deteccao de agentes.
type DetectOptions struct {
	// ProjectDir e o diretorio do projeto corrente (para FileDetector como sinal adicional).
	// Vazio = usar apenas binario no PATH e dirs de config.
	ProjectDir string
}

// AgentDetector detecta quais agentes/CLIs de IA estao presentes no ambiente.
// Combina tres sinais: binario ACP no PATH, diretorios de configuracao conhecidos
// (~/.claude, ~/.codex, ~/.gemini) e arquivos de projeto (via FileDetector).
// ADR-019.
type AgentDetector interface {
	// Detect retorna os agentes presentes. Nunca executa binarios — LookPath apenas.
	Detect(ctx context.Context, opts DetectOptions) ([]skills.Tool, error)
}

// agentEntry mapeia um Tool ao seu binario canonico (extraido de specs) e aos
// diretorios de config conhecidos no home dir do usuario.
type agentEntry struct {
	tool     skills.Tool
	command  string   // binario canonico da Spec
	homeDirs []string // subdirs em $HOME que indicam presenca do agente
}

// allEntries constroi as entradas a partir das Specs canonicas (ADR-019: reusar nomes
// de comando das specs, sem duplicar literais).
func allEntries() []agentEntry {
	return []agentEntry{
		{
			tool:     skills.ToolClaude,
			command:  specs.Claude().Command,
			homeDirs: []string{".claude"},
		},
		{
			tool:     skills.ToolCodex,
			command:  specs.Codex().Command,
			homeDirs: []string{".codex"},
		},
		{
			tool:     skills.ToolGemini,
			command:  specs.Gemini().Command,
			homeDirs: []string{".gemini"},
		},
		{
			tool:     skills.ToolCopilot,
			command:  specs.Copilot().Command,
			homeDirs: []string{".copilot", filepath.Join(".github", "copilot")},
		},
	}
}

// isDirOnDisk verifica se o path existe e e um diretorio no filesystem real.
// Usado apenas para dirs de config em $HOME (sinal secundario).
func isDirOnDisk(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// BinaryAgentDetector implementa AgentDetector usando LookPath e dirs de config.
type BinaryAgentDetector struct {
	lookPath LookPather
	homeDir  HomeDir
	fileDet  *FileDetector // sinal adicional: arquivos de projeto (pode ser nil)
}

// NewBinaryAgentDetector cria um detector com dependencias injetadas.
// fileDet pode ser nil — nesse caso o sinal de arquivos de projeto e omitido.
func NewBinaryAgentDetector(lp LookPather, hd HomeDir, fd *FileDetector) *BinaryAgentDetector {
	return &BinaryAgentDetector{
		lookPath: lp,
		homeDir:  hd,
		fileDet:  fd,
	}
}

// Detect retorna os Tools presentes no ambiente.
// Ordem de verificacao por Tool:
//  1. Binario ACP no PATH (LookPath) — sinal primario.
//  2. Dir de config conhecido em $HOME (ex: ~/.claude) — sinal secundario.
//  3. Arquivos de projeto via FileDetector (ex: CLAUDE.md, .claude/) — sinal terciario.
//
// Basta qualquer sinal para incluir o Tool. Duplicatas sao eliminadas.
// Nunca executa binarios — LookPath apenas (R-SEC-001).
func (d *BinaryAgentDetector) Detect(ctx context.Context, opts DetectOptions) ([]skills.Tool, error) {
	homeRoot, homeErr := d.homeDir.UserHomeDir()
	// homeErr nao aborta — global e opt-in; sem $HOME degrada sem erro fatal.

	// Sinal de arquivos de projeto (FileDetector).
	var fileToolSet map[skills.Tool]bool
	if d.fileDet != nil && opts.ProjectDir != "" {
		fileTools := d.fileDet.DetectTools(opts.ProjectDir)
		fileToolSet = make(map[skills.Tool]bool, len(fileTools))
		for _, t := range fileTools {
			fileToolSet[t] = true
		}
	}

	seen := make(map[skills.Tool]bool)
	var result []skills.Tool

	for _, entry := range allEntries() {
		if seen[entry.tool] {
			continue
		}

		// Sinal 1: binario no PATH — sinal primario (R-SEC-001: sem executar).
		if _, err := d.lookPath.LookPath(entry.command); err == nil {
			seen[entry.tool] = true
			result = append(result, entry.tool)
			continue
		}

		// Sinal 2: dir de config em $HOME — sinal secundario.
		if homeErr == nil {
			for _, sub := range entry.homeDirs {
				full := filepath.Join(homeRoot, sub)
				if isDirOnDisk(full) {
					seen[entry.tool] = true
					result = append(result, entry.tool)
					break
				}
			}
			if seen[entry.tool] {
				continue
			}
		}

		// Sinal 3: arquivo de projeto — sinal terciario.
		if fileToolSet[entry.tool] {
			seen[entry.tool] = true
			result = append(result, entry.tool)
		}
	}

	return result, nil
}
