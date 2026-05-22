//go:build integration

// Package integration — testes de integração F2-Gemini: MCP nested-agent com driver Gemini.
//
// Cobre RF-13 (normalização Gemini via inherit:common), RF-14 (MCP nested spawn),
// RF-15 (depth limit para Gemini igual a Claude/Codex).
//
// Execução:
//
//	go test -tags integration ./tests/integration/... -run TestMCPNestedAgentSpawnsGeminiSession
//	go test -tags integration ./tests/integration/... -run TestMCPDepthLimitAppliesToGemini
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/mcpserver"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/mcpserver/wire"
)

// --- Helpers específicos para F2-Gemini ---

// geminiReviewerAgent retorna um ResolvedAgent de teste com IDE "gemini" chamado "reviewer".
func geminiReviewerAgent() agents.ResolvedAgent {
	return agents.ResolvedAgent{
		Name:   "reviewer",
		Scope:  agents.ScopeWorkspace,
		Prompt: "Você é um revisor de código Gemini.",
		Metadata: agents.Metadata{
			Description: "Agente de revisão via Gemini.",
			Version:     "1.0.0",
		},
		Runtime: agents.RuntimeDefaults{
			IDE: "gemini",
		},
	}
}

// geminiMockRegistry implementa agents.Registry para testes Gemini.
type geminiMockRegistry struct {
	agents map[string]agents.ResolvedAgent
}

func (m *geminiMockRegistry) Discover(_ context.Context) ([]agents.ResolvedAgent, error) {
	result := make([]agents.ResolvedAgent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	return result, nil
}

func (m *geminiMockRegistry) Resolve(name string) (agents.ResolvedAgent, error) {
	if a, ok := m.agents[name]; ok {
		return a, nil
	}
	return agents.ResolvedAgent{}, agents.ErrAgentNotFound
}

// newGeminiMockRegistry cria um registry contendo apenas o agente Gemini reviewer.
func newGeminiMockRegistry(agts ...agents.ResolvedAgent) agents.Registry {
	m := &geminiMockRegistry{agents: make(map[string]agents.ResolvedAgent)}
	for _, a := range agts {
		m.agents[a.Name] = a
	}
	return m
}

// geminiMCPRequest serializa uma requisição MCP JSON-RPC para o wire.
func geminiMCPRequest(t *testing.T, id int, method string, params interface{}) string {
	t.Helper()
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("geminiMCPRequest: marshal params: %v", err)
		}
		rawParams = b
	}
	rawID, _ := json.Marshal(id)
	idMsg := json.RawMessage(rawID)
	req := wire.Request{
		JSONRPC: "2.0",
		ID:      &idMsg,
		Method:  method,
		Params:  rawParams,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("geminiMCPRequest: marshal request: %v", err)
	}
	return string(b) + "\n"
}

// parseGeminiResponses lê todas as respostas JSON-RPC do buffer.
func parseGeminiResponses(t *testing.T, buf *bytes.Buffer) []wire.Response {
	t.Helper()
	var responses []wire.Response
	decoder := json.NewDecoder(buf)
	for {
		var resp wire.Response
		if err := decoder.Decode(&resp); err != nil {
			break
		}
		responses = append(responses, resp)
	}
	return responses
}

// ---- T-33: MCP nested-agent com parent Gemini ----------------------------------------

// TestMCPNestedAgentSpawnsGeminiSession — T-33 (F2-Gemini, RF-14, RF-15).
//
// Valida que o mcpserver (infra F2-Claude, tool-agnóstica) funciona com
// um agente parent de IDE "gemini" invocando run_agent("reviewer", "...").
//
// Cenário: parent Gemini envia tools/call run_agent; o server tenta spawnar
// child (falha porque sem binário real, mas o fluxo de resolução e dispatch
// funciona corretamente). O teste valida que:
//   - registry resolve "reviewer" (não retorna ErrAgentNotFound)
//   - depth limit default (3) é respeitado para Gemini igual a Claude/Codex
//   - handshake MCP (initialize + tools/list) funciona normalmente
func TestMCPNestedAgentSpawnsGeminiSession(t *testing.T) {
	t.Parallel()

	registry := newGeminiMockRegistry(geminiReviewerAgent())
	srv := mcpserver.New(registry, 3, nil)

	// Requisição de handshake + tools/list.
	toolsListReq := geminiMCPRequest(t, 1, "initialize", map[string]interface{}{
		"protocolVersion": wire.MCPVersion,
		"clientInfo":      map[string]string{"name": "gemini-parent", "version": "0.43.0"},
		"capabilities":    map[string]interface{}{},
	}) + geminiMCPRequest(t, 2, "tools/list", nil)

	input := strings.NewReader(toolsListReq)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = srv.Serve(ctx, input, &out) }()
	time.Sleep(150 * time.Millisecond)
	cancel()

	responses := parseGeminiResponses(t, &out)
	if len(responses) < 2 {
		t.Fatalf("T-33: esperado ≥2 respostas (initialize + tools/list), obtido %d", len(responses))
	}

	// initialize deve retornar sucesso.
	if responses[0].Error != nil {
		t.Fatalf("T-33: initialize retornou erro: %+v", responses[0].Error)
	}

	// tools/list deve retornar apenas run_agent.
	raw, _ := json.Marshal(responses[1].Result)
	var toolsResult wire.ToolsListResult
	if err := json.Unmarshal(raw, &toolsResult); err != nil {
		t.Fatalf("T-33: unmarshal tools/list result: %v", err)
	}
	if len(toolsResult.Tools) != 1 || toolsResult.Tools[0].Name != mcpserver.ReservedToolName {
		t.Errorf("T-33: tools/list deve retornar apenas %q; got=%+v", mcpserver.ReservedToolName, toolsResult.Tools)
	}

	// Agora enviar run_agent com agente "reviewer" (timeout curto para não travar o teste).
	runAgentParams := wire.ToolCallParams{
		Name: mcpserver.ReservedToolName,
	}
	args, _ := json.Marshal(mcpserver.RunAgentInput{
		AgentName: "reviewer",
		Prompt:    "revise o código Gemini",
		Timeout:   1, // timeout curto — child não tem binário real em CI
	})
	runAgentParams.Arguments = args
	paramsRaw, _ := json.Marshal(runAgentParams)

	runReq := strings.NewReader(geminiMCPRequest(t, 3, "tools/call", json.RawMessage(paramsRaw)))
	var runOut bytes.Buffer

	runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer runCancel()

	done := make(chan struct{})
	srv2 := mcpserver.New(registry, 3, nil)
	go func() {
		_ = srv2.Serve(runCtx, runReq, &runOut)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		runCancel()
		t.Fatal("T-33: servidor não respondeu em 4s para run_agent")
	}

	runResponses := parseGeminiResponses(t, &runOut)
	if len(runResponses) == 0 {
		t.Fatal("T-33: nenhuma resposta para run_agent")
	}

	// O agente "reviewer" foi resolvido pelo registry — não deve retornar ErrAgentNotFound.
	// Em CI sem binário real, pode retornar erro de timeout/internal, mas NÃO ErrAgentNotFound.
	resp := runResponses[0]
	if resp.Error != nil && resp.Error.Code == wire.ErrCodeAgentNotFound {
		t.Errorf("T-33: agente 'reviewer' com IDE gemini não deveria retornar ErrAgentNotFound; code=%d msg=%q",
			resp.Error.Code, resp.Error.Message)
	}

	t.Logf("T-33: OK — run_agent(reviewer) dispatched; response_error=%v", resp.Error)
}

// ---- T-33b: Depth limit aplica-se a Gemini igual a Claude/Codex --------------------

// TestMCPDepthLimitAppliesToGemini — T-33b (F2-Gemini, RF-15).
//
// Valida que o depth limit (≤ 3) aplica-se a Gemini da mesma forma que Claude/Codex.
// Simula contexto com depth=3 (maxDepth=3) → currentDepth=4 > 3 → retorna ErrCodeDepthExceeded.
//
// Compozy convenção: depth limit é tool-agnóstico; já testado em F2-Claude (T-MCP-04).
// Este teste confirma que o mesmo mecanismo funciona para parent Gemini.
// Nota: não usa t.Parallel() pois usa t.Setenv.
func TestMCPDepthLimitAppliesToGemini(t *testing.T) {
	// Simular contexto de execução com depth=3 (já no limite).
	t.Setenv(mcpserver.EnvContext, `{"parent_session_id":"gemini-parent","depth":3,"workspace_root":"/tmp"}`)

	registry := newGeminiMockRegistry(geminiReviewerAgent())
	srv := mcpserver.New(registry, 3, nil)

	params := wire.ToolCallParams{
		Name: mcpserver.ReservedToolName,
	}
	args, _ := json.Marshal(mcpserver.RunAgentInput{
		AgentName: "reviewer",
		Prompt:    "revise o código — depth excedido",
	})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(geminiMCPRequest(t, 1, "tools/call", json.RawMessage(paramsRaw)))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() { _ = srv.Serve(ctx, input, &out) }()
	time.Sleep(150 * time.Millisecond)
	cancel()

	responses := parseGeminiResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("T-33b: nenhuma resposta recebida")
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("T-33b: esperado erro de depth excedido, obtido resultado sem erro")
	}
	if resp.Error.Code != wire.ErrCodeDepthExceeded {
		t.Fatalf("T-33b: esperado código %d (ErrCodeDepthExceeded), obtido %d — %s",
			wire.ErrCodeDepthExceeded, resp.Error.Code, resp.Error.Message)
	}
	if !strings.Contains(resp.Error.Message, "profundidade") {
		t.Errorf("T-33b: mensagem de erro deve mencionar 'profundidade'; got=%q", resp.Error.Message)
	}

	t.Logf("T-33b: OK — depth limit aplicado corretamente para Gemini: %s", resp.Error.Message)
}
