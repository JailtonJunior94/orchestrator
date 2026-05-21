package agents

import (
	"fmt"
	"strings"
	"testing"
)

func makeAgent(name, description, version string) ResolvedAgent {
	return ResolvedAgent{
		Name: name,
		Metadata: Metadata{
			Description: description,
			Version:     version,
		},
	}
}

// T-15: catalogo de 3 entradas -> bloco ordenado lex; agente ativo marcado.
func TestBuildAgentBlocks_OrderedAndActive(t *testing.T) {
	active := ResolvedAgent{
		Name: "beta-agent",
		Metadata: Metadata{
			Description: "Beta agent description",
			Version:     "1.0.0",
		},
		Runtime: RuntimeDefaults{
			IDE:   "claude",
			Model: "claude-opus-4-7",
		},
	}
	catalog := []ResolvedAgent{
		makeAgent("zeta-agent", "Zeta agent", "1.0.0"),
		makeAgent("alpha-agent", "Alpha agent", "2.0.0"),
		makeAgent("beta-agent", "Beta agent description", "1.0.0"),
	}

	meta, cat := BuildAgentBlocks(&active, catalog)

	// Verificar bloco metadata.
	if !strings.Contains(meta, "### Agente Ativo") {
		t.Error("T-15: metadata deve conter '### Agente Ativo'")
	}
	if !strings.Contains(meta, "beta-agent") {
		t.Error("T-15: metadata deve conter o nome do agente ativo")
	}
	if !strings.Contains(meta, "Beta agent description") {
		t.Error("T-15: metadata deve conter a descricao")
	}
	if !strings.Contains(meta, "1.0.0") {
		t.Error("T-15: metadata deve conter a versao")
	}
	if !strings.Contains(meta, "claude") {
		t.Error("T-15: metadata deve conter o IDE no runtime")
	}

	// Verificar bloco catalogo.
	if !strings.Contains(cat, "### Agentes Disponiveis") {
		t.Error("T-15: catalogo deve conter '### Agentes Disponiveis'")
	}
	if !strings.Contains(cat, "`beta-agent` [active]") {
		t.Error("T-15: agente ativo deve ser marcado com [active]")
	}

	// Verificar ordem lexicografica.
	alphaPos := strings.Index(cat, "alpha-agent")
	betaPos := strings.Index(cat, "beta-agent")
	zetaPos := strings.Index(cat, "zeta-agent")
	if alphaPos < 0 || betaPos < 0 || zetaPos < 0 {
		t.Fatal("T-15: todos os agentes devem aparecer no catalogo")
	}
	if alphaPos >= betaPos || betaPos >= zetaPos {
		t.Errorf("T-15: ordem incorreta — alpha=%d, beta=%d, zeta=%d", alphaPos, betaPos, zetaPos)
	}
}

// T-16: catalogo com 250 entradas -> truncado para 200.
func TestBuildAgentBlocks_TruncatesAt200(t *testing.T) {
	active := makeAgent("agent-000", "Active", "1.0.0")

	catalog := make([]ResolvedAgent, 250)
	for i := range catalog {
		name := fmt.Sprintf("agent-%03d", i)
		catalog[i] = makeAgent(name, "desc", "1.0.0")
	}

	_, cat := BuildAgentBlocks(&active, catalog)

	// Contar entradas no catalogo.
	count := strings.Count(cat, "\n- `")
	if count != catalogLimit {
		t.Errorf("T-16: catalogo deve ter %d entradas, tem %d", catalogLimit, count)
	}
}

// Extra: agente ativo nil nao causa panic.
func TestBuildAgentBlocks_NilActiveNoPanic(t *testing.T) {
	catalog := []ResolvedAgent{
		makeAgent("alpha-agent", "Alpha", "1.0.0"),
	}
	// Nao deve entrar em panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BuildAgentBlocks com active nil causou panic: %v", r)
		}
	}()
	BuildAgentBlocks(nil, catalog)
}

// Extra: catalogo vazio -> bloco de catalogo com apenas o header.
func TestBuildAgentBlocks_EmptyCatalog(t *testing.T) {
	active := makeAgent("my-agent", "My agent", "1.0.0")
	meta, cat := BuildAgentBlocks(&active, []ResolvedAgent{})

	if !strings.Contains(meta, "### Agente Ativo") {
		t.Error("metadata deveria ter header mesmo com catalogo vazio")
	}
	if !strings.Contains(cat, "### Agentes Disponiveis") {
		t.Error("catalogo deveria ter header mesmo com catalogo vazio")
	}
	// Nenhuma entrada de agente no catalogo.
	if strings.Contains(cat, "`") {
		t.Error("catalogo vazio nao deve ter entradas de agentes")
	}
}

// Extra: runtime parcial -> linha de runtime omite campos vazios.
func TestBuildAgentBlocks_RuntimePartial(t *testing.T) {
	active := ResolvedAgent{
		Name: "partial-agent",
		Metadata: Metadata{
			Description: "Partial",
			Version:     "1.0.0",
		},
		Runtime: RuntimeDefaults{
			IDE: "claude",
			// Model, ReasoningEffort e AccessMode vazios.
		},
	}

	meta, _ := BuildAgentBlocks(&active, []ResolvedAgent{active})

	if !strings.Contains(meta, "claude") {
		t.Error("metadata deve conter IDE mesmo quando outros campos de runtime estao vazios")
	}
}

// Extra: runtime totalmente vazio -> linha de runtime exibe "-".
func TestBuildAgentBlocks_RuntimeEmpty(t *testing.T) {
	active := ResolvedAgent{
		Name: "empty-rt",
		Metadata: Metadata{
			Description: "No runtime",
			Version:     "1.0.0",
		},
	}

	meta, _ := BuildAgentBlocks(&active, []ResolvedAgent{active})

	if !strings.Contains(meta, "- **Runtime**: -") {
		t.Errorf("metadata com runtime vazio deve exibir '-', got:\n%s", meta)
	}
}
