package detect

import (
	"context"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// fakeLookPath simula exec.LookPath com um conjunto de binarios "presentes".
type fakeLookPath struct {
	present map[string]bool
}

func (f fakeLookPath) LookPath(file string) (string, error) {
	if f.present[file] {
		return "/usr/local/bin/" + file, nil
	}
	return "", errors.New("nao encontrado")
}

// fakeHomeDir retorna um home dir fixo para testes.
type fakeHomeDir struct {
	home string
	err  error
}

func (f fakeHomeDir) UserHomeDir() (string, error) {
	return f.home, f.err
}

// --- Testes ---

func TestBinaryAgentDetector_BinaryInPath(t *testing.T) {
	t.Parallel()

	lp := fakeLookPath{present: map[string]bool{
		"claude-agent-acp": true,
		"codex-acp":        true,
	}}
	hd := fakeHomeDir{home: "/nonexistent-home-dir-xyz"}
	det := NewBinaryAgentDetector(lp, hd, nil)

	got, err := det.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("Detect retornou erro inesperado: %v", err)
	}

	toolSet := toSet(got)
	if !toolSet[skills.ToolClaude] {
		t.Error("esperava ToolClaude detectado via binario")
	}
	if !toolSet[skills.ToolCodex] {
		t.Error("esperava ToolCodex detectado via binario")
	}
	if toolSet[skills.ToolGemini] {
		t.Error("nao esperava ToolGemini — binario ausente")
	}
	if toolSet[skills.ToolCopilot] {
		t.Error("nao esperava ToolCopilot — binario ausente")
	}
}

func TestBinaryAgentDetector_FileProjectOnly(t *testing.T) {
	t.Parallel()

	// Nenhum binario no PATH.
	lp := fakeLookPath{present: map[string]bool{}}
	// Home dir inexistente => sem sinal de dir de config.
	hd := fakeHomeDir{home: "/nonexistent-home-xyz"}
	// FakeFileSystem com indicador de Claude via CLAUDE.md.
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/CLAUDE.md"] = []byte("# Claude")
	fileDet := NewFileDetector(ffs)
	det := NewBinaryAgentDetector(lp, hd, fileDet)

	got, err := det.Detect(context.Background(), DetectOptions{ProjectDir: "/project"})
	if err != nil {
		t.Fatalf("Detect retornou erro: %v", err)
	}

	toolSet := toSet(got)
	if !toolSet[skills.ToolClaude] {
		t.Error("esperava ToolClaude detectado via arquivo de projeto")
	}
	if len(got) != 1 {
		t.Errorf("esperava exatamente 1 ferramenta, got %d: %v", len(got), got)
	}
}

func TestBinaryAgentDetector_EmptyRepo_NoBinaries(t *testing.T) {
	t.Parallel()

	// Sem binarios, sem home existente, sem arquivos de projeto.
	lp := fakeLookPath{present: map[string]bool{}}
	hd := fakeHomeDir{home: "/nonexistent-home-xyz"}
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/empty-project"] = true
	fileDet := NewFileDetector(ffs)
	det := NewBinaryAgentDetector(lp, hd, fileDet)

	got, err := det.Detect(context.Background(), DetectOptions{ProjectDir: "/empty-project"})
	if err != nil {
		t.Fatalf("Detect retornou erro: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("esperava vazio, got %v", got)
	}
}

func TestBinaryAgentDetector_HomeDirError_DegradesGracefully(t *testing.T) {
	t.Parallel()

	// UserHomeDir retorna erro => sinal de config dir e omitido, sem falha fatal.
	lp := fakeLookPath{present: map[string]bool{}}
	hd := fakeHomeDir{err: errors.New("HOME nao definido")}
	det := NewBinaryAgentDetector(lp, hd, nil)

	got, err := det.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("Detect nao deve retornar erro quando HOME ausente: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("esperava vazio sem binarios e sem HOME, got %v", got)
	}
}

func TestBinaryAgentDetector_AllBinaries(t *testing.T) {
	t.Parallel()

	lp := fakeLookPath{present: map[string]bool{
		"claude-agent-acp": true,
		"codex-acp":        true,
		"gemini":           true,
		"copilot":          true,
	}}
	hd := fakeHomeDir{home: "/nonexistent-home-xyz"}
	det := NewBinaryAgentDetector(lp, hd, nil)

	got, err := det.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("Detect retornou erro: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("esperava 4 ferramentas, got %d: %v", len(got), got)
	}
}

func TestBinaryAgentDetector_NoDuplicates(t *testing.T) {
	t.Parallel()

	// Claude via binario E via arquivo de projeto => deve aparecer uma unica vez.
	lp := fakeLookPath{present: map[string]bool{"claude-agent-acp": true}}
	hd := fakeHomeDir{home: "/nonexistent-home-xyz"}
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/CLAUDE.md"] = []byte("# Claude")
	fileDet := NewFileDetector(ffs)
	det := NewBinaryAgentDetector(lp, hd, fileDet)

	got, err := det.Detect(context.Background(), DetectOptions{ProjectDir: "/project"})
	if err != nil {
		t.Fatalf("Detect retornou erro: %v", err)
	}

	count := 0
	for _, t := range got {
		if t == skills.ToolClaude {
			count++
		}
	}
	if count != 1 {
		t.Errorf("ToolClaude deveria aparecer exatamente 1 vez, got %d", count)
	}
}

func TestAllEntries_CommandsMatchSpecs(t *testing.T) {
	t.Parallel()

	// Verifica que allEntries reusa nomes de comando das Specs (sem duplicar literais).
	entries := allEntries()
	if len(entries) != 4 {
		t.Fatalf("esperava 4 entries (claude, codex, gemini, copilot), got %d", len(entries))
	}
	for _, e := range entries {
		if e.command == "" {
			t.Errorf("entry %s tem command vazio", e.tool)
		}
		if e.tool == "" {
			t.Error("entry com tool vazio")
		}
	}
}

// toSet converte slice de Tool em mapa para lookup O(1).
func toSet(tools []skills.Tool) map[skills.Tool]bool {
	m := make(map[skills.Tool]bool, len(tools))
	for _, t := range tools {
		m[t] = true
	}
	return m
}
