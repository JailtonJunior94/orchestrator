package agents_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// validAgentMD retorna um AGENT.md minimo valido com o nome indicado.
func validAgentMD(name string) string {
	return "---\nname: " + name + "\ndescription: Agente de teste\nversion: 1.0.0\n---\n\nCorpo do agente.\n"
}

// setupFakeFS retorna um FakeFileSystem com agentes nos escopos indicados.
// globalAgents e workspaceAgents sao listas de nomes de agentes a criar.
func setupFakeFS(globalRoot, workspaceRoot string, globalAgents, workspaceAgents []string) *fs.FakeFileSystem {
	fake := fs.NewFakeFileSystem()

	for _, name := range globalAgents {
		path := globalRoot + "/.ai-harness/agents/" + name + "/AGENT.md"
		fake.Files[path] = []byte(validAgentMD(name))
	}

	for _, name := range workspaceAgents {
		path := workspaceRoot + "/.ai-harness/agents/" + name + "/AGENT.md"
		fake.Files[path] = []byte(validAgentMD(name))
	}

	return fake
}

// --- Testes de discoverAgents ---

// T-01: nenhum agente em nenhum escopo → discoverAgents retorna nil, sem erro.
func TestDiscoverAgents_T01_Vazio(t *testing.T) {
	t.Parallel()
	fake := fs.NewFakeFileSystem()

	result, err := agents.ExportDiscoverAgents(fake, agents.ScopeGlobal, "/home/user")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("esperado 0 agentes, obteve %d", len(result))
	}
}

// T-02: apenas global → lista apenas global, Scope=ScopeGlobal.
func TestDiscoverAgents_T02_ApenasGlobal(t *testing.T) {
	t.Parallel()
	fake := setupFakeFS("/home/user", "/workspace", []string{"agente-alpha"}, nil)

	result, err := agents.ExportDiscoverAgents(fake, agents.ScopeGlobal, "/home/user")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("esperado 1 agente, obteve %d", len(result))
	}
	if result[0].Name != "agente-alpha" {
		t.Errorf("nome esperado %q, obteve %q", "agente-alpha", result[0].Name)
	}
	if result[0].Scope != agents.ScopeGlobal {
		t.Errorf("scope esperado ScopeGlobal, obteve %v", result[0].Scope)
	}
	if result[0].Path == "" {
		t.Error("path nao pode ser vazio")
	}
}

// T-03: apenas workspace → lista apenas workspace, Scope=ScopeWorkspace.
func TestDiscoverAgents_T03_ApenasWorkspace(t *testing.T) {
	t.Parallel()
	fake := setupFakeFS("/home/user", "/workspace", nil, []string{"agente-beta"})

	result, err := agents.ExportDiscoverAgents(fake, agents.ScopeWorkspace, "/workspace")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("esperado 1 agente, obteve %d", len(result))
	}
	if result[0].Name != "agente-beta" {
		t.Errorf("nome esperado %q, obteve %q", "agente-beta", result[0].Name)
	}
	if result[0].Scope != agents.ScopeWorkspace {
		t.Errorf("scope esperado ScopeWorkspace, obteve %v", result[0].Scope)
	}
}

// T-04: colisao global+workspace → workspace prevalece via mergeWithShadowing.
func TestMergeWithShadowing_T04_Colisao(t *testing.T) {
	t.Parallel()

	global := []agents.ResolvedAgent{
		{Name: "agente-x", Scope: agents.ScopeGlobal},
		{Name: "agente-y", Scope: agents.ScopeGlobal},
	}
	workspace := []agents.ResolvedAgent{
		{Name: "agente-x", Scope: agents.ScopeWorkspace}, // colide com global
	}

	merged, shadowed := agents.ExportMergeWithShadowing(global, workspace)

	// agente-x deve aparecer apenas uma vez (workspace).
	found := map[string]agents.Scope{}
	for _, a := range merged {
		found[a.Name] = a.Scope
	}

	if scope, ok := found["agente-x"]; !ok {
		t.Error("agente-x deve estar em merged")
	} else if scope != agents.ScopeWorkspace {
		t.Errorf("agente-x deve ter ScopeWorkspace, obteve %v", scope)
	}

	if _, ok := found["agente-y"]; !ok {
		t.Error("agente-y deve estar em merged (sem colisao)")
	}

	if len(merged) != 2 {
		t.Errorf("merged deve ter 2 agentes, obteve %d", len(merged))
	}

	// Global ofuscado deve aparecer em shadowed.
	if len(shadowed) != 1 {
		t.Fatalf("shadowed deve ter 1 agente, obteve %d", len(shadowed))
	}
	if shadowed[0].Name != "agente-x" {
		t.Errorf("shadowed[0] deve ser agente-x, obteve %q", shadowed[0].Name)
	}
	if shadowed[0].Scope != agents.ScopeGlobal {
		t.Errorf("shadowed[0] deve ter ScopeGlobal, obteve %v", shadowed[0].Scope)
	}
}

// T-09: name no frontmatter != dirname → erro RF-06 propagado e agente ignorado, demais continuam.
func TestDiscoverAgents_T09_NameDirMismatch(t *testing.T) {
	t.Parallel()
	fake := fs.NewFakeFileSystem()

	// Agente com name divergente do diretório.
	const mismatchPath = "/home/user/.ai-harness/agents/meu-agente/AGENT.md"
	fake.Files[mismatchPath] = []byte("---\nname: nome-diferente\ndescription: Teste\nversion: 1.0.0\n---\n")

	// Agente valido para garantir que a descoberta continua apos o erro.
	const validPath = "/home/user/.ai-harness/agents/agente-valido/AGENT.md"
	fake.Files[validPath] = []byte(validAgentMD("agente-valido"))

	result, err := agents.ExportDiscoverAgents(fake, agents.ScopeGlobal, "/home/user")

	// Deve retornar erro (parcial) mas tambem o agente valido.
	if err == nil {
		t.Error("esperado erro por RF-06, obteve nil")
	}
	if !errors.Is(err, agents.ErrNameDirMismatch) {
		t.Errorf("erro esperado ErrNameDirMismatch, obteve: %v", err)
	}

	// O agente valido deve aparecer mesmo com erro parcial.
	if len(result) != 1 {
		t.Errorf("esperado 1 agente valido, obteve %d", len(result))
	}
	if len(result) > 0 && result[0].Name != "agente-valido" {
		t.Errorf("agente valido esperado %q, obteve %q", "agente-valido", result[0].Name)
	}
}

// TestDiscoverAgents_FrontmatterInvalido_ContinuaDescoberta verifica que frontmatter invalido
// em um agente nao interrompe a descoberta dos demais.
func TestDiscoverAgents_FrontmatterInvalido_ContinuaDescoberta(t *testing.T) {
	t.Parallel()
	fake := fs.NewFakeFileSystem()

	// Agente com frontmatter invalido (sem description obrigatorio).
	fake.Files["/home/user/.ai-harness/agents/invalido/AGENT.md"] = []byte(
		"---\nname: invalido\nversion: 1.0.0\n---\n",
	)

	// Agente valido.
	fake.Files["/home/user/.ai-harness/agents/valido/AGENT.md"] = []byte(validAgentMD("valido"))

	result, err := agents.ExportDiscoverAgents(fake, agents.ScopeGlobal, "/home/user")

	// Deve haver erro parcial.
	if err == nil {
		t.Error("esperado erro parcial por frontmatter invalido")
	}

	// Mas o agente valido deve ser retornado.
	if len(result) != 1 || result[0].Name != "valido" {
		t.Errorf("esperado agente 'valido' em resultado, obteve: %v", result)
	}
}

// TestMergeWithShadowing_SemColisao verifica que sem colisao todos os agentes aparecem em merged.
func TestMergeWithShadowing_SemColisao(t *testing.T) {
	t.Parallel()

	global := []agents.ResolvedAgent{
		{Name: "agente-g", Scope: agents.ScopeGlobal},
	}
	workspace := []agents.ResolvedAgent{
		{Name: "agente-w", Scope: agents.ScopeWorkspace},
	}

	merged, shadowed := agents.ExportMergeWithShadowing(global, workspace)

	if len(merged) != 2 {
		t.Errorf("merged deve ter 2 agentes, obteve %d", len(merged))
	}
	if len(shadowed) != 0 {
		t.Errorf("shadowed deve ser vazio, obteve %d", len(shadowed))
	}
}

// TestMergeWithShadowing_OrdemLexicografica verifica que merged e retornado em ordem lexicografica.
func TestMergeWithShadowing_OrdemLexicografica(t *testing.T) {
	t.Parallel()

	global := []agents.ResolvedAgent{
		{Name: "zebra", Scope: agents.ScopeGlobal},
		{Name: "alpha", Scope: agents.ScopeGlobal},
	}

	merged, _ := agents.ExportMergeWithShadowing(global, nil)

	if len(merged) != 2 {
		t.Fatalf("esperado 2 agentes, obteve %d", len(merged))
	}
	if merged[0].Name != "alpha" || merged[1].Name != "zebra" {
		t.Errorf("ordem lexicografica esperada [alpha, zebra], obteve [%s, %s]", merged[0].Name, merged[1].Name)
	}
}
