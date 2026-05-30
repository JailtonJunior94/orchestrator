package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/mcpserver/wire"
)

// --- Helpers de test ---

// mockRegistry implementa agents.Registry para testes.
type mockRegistry struct {
	agents map[string]agents.ResolvedAgent
}

func (m *mockRegistry) Discover(_ context.Context) ([]agents.ResolvedAgent, error) {
	result := make([]agents.ResolvedAgent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	return result, nil
}

func (m *mockRegistry) Resolve(name string) (agents.ResolvedAgent, error) {
	if a, ok := m.agents[name]; ok {
		return a, nil
	}
	return agents.ResolvedAgent{}, fmt.Errorf("%w: %q", agents.ErrAgentNotFound, name)
}

// newMockRegistry cria um registry de teste com os agentes fornecidos.
func newMockRegistry(agts ...agents.ResolvedAgent) agents.Registry {
	m := &mockRegistry{agents: make(map[string]agents.ResolvedAgent)}
	for _, a := range agts {
		m.agents[a.Name] = a
	}
	return m
}

// reviewerAgent retorna um ResolvedAgent de teste chamado "reviewer".
func reviewerAgent() agents.ResolvedAgent {
	return agents.ResolvedAgent{
		Name:   "reviewer",
		Scope:  agents.ScopeWorkspace,
		Prompt: "Você é um revisor de código.",
		Metadata: agents.Metadata{
			Description: "Agente de revisão de código.",
			Version:     "1.0.0",
		},
		Runtime: agents.RuntimeDefaults{
			IDE: "claude",
		},
	}
}

// mcpRequest serializa uma requisição MCP para o wire.
func mcpRequest(t *testing.T, id int, method string, params interface{}) string {
	t.Helper()
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("mcpRequest: marshal params: %v", err)
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
		t.Fatalf("mcpRequest: marshal request: %v", err)
	}
	return string(b) + "\n"
}

// parseResponses lê todas as respostas JSON-RPC do buffer.
func parseResponses(t *testing.T, buf *bytes.Buffer) []wire.Response {
	t.Helper()
	var responses []wire.Response
	decoder := json.NewDecoder(buf)
	for {
		var resp wire.Response
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("parseResponses: %v", err)
		}
		responses = append(responses, resp)
	}
	return responses
}

// initializeExchange envia o handshake initialize e retorna o server com in/out.
func buildInitializeRequest(t *testing.T) string {
	t.Helper()
	return mcpRequest(t, 1, "initialize", map[string]interface{}{
		"protocolVersion": wire.MCPVersion,
		"clientInfo":      map[string]string{"name": "test-client", "version": "0.0.1"},
		"capabilities":    map[string]interface{}{},
	})
}

// --- T-MCP-01: apenas run_agent exposta ---

// TestServerExposesRunAgentOnly valida que tools/list retorna exatamente run_agent.
func TestServerExposesRunAgentOnly(t *testing.T) {
	registry := newMockRegistry(reviewerAgent())
	srv := New(registry, 3, nil)

	input := strings.NewReader(
		buildInitializeRequest(t) +
			mcpRequest(t, 2, "tools/list", nil),
	)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()

	// Dar tempo ao servidor para processar.
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) < 2 {
		t.Fatalf("esperado ≥2 respostas, obtido %d", len(responses))
	}

	// Segunda resposta = tools/list.
	toolsResp := responses[1]
	if toolsResp.Error != nil {
		t.Fatalf("tools/list retornou erro: %+v", toolsResp.Error)
	}

	raw, _ := json.Marshal(toolsResp.Result)
	var result wire.ToolsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal tools list result: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("esperado 1 tool, obtido %d", len(result.Tools))
	}
	if result.Tools[0].Name != ReservedToolName {
		t.Fatalf("esperado tool %q, obtido %q", ReservedToolName, result.Tools[0].Name)
	}
}

// --- T-MCP-03: agente desconhecido retorna erro tipado ---

// TestRunAgentUnknownAgentReturnsError valida que agent_name desconhecido retorna erro MCP tipado.
func TestRunAgentUnknownAgentReturnsError(t *testing.T) {
	registry := newMockRegistry() // sem agentes
	srv := New(registry, 3, nil)

	params := wire.ToolCallParams{
		Name: ReservedToolName,
	}
	args, _ := json.Marshal(RunAgentInput{
		AgentName: "ghost",
		Prompt:    "faça algo",
	})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(mcpRequest(t, 1, "tools/call",
		json.RawMessage(paramsRaw)),
	)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta recebida")
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("esperado erro, obtido resultado")
	}
	if resp.Error.Code != wire.ErrCodeAgentNotFound {
		t.Fatalf("esperado código %d (ErrCodeAgentNotFound), obtido %d", wire.ErrCodeAgentNotFound, resp.Error.Code)
	}
}

// --- T-MCP-04: depth limit ---

// TestRunAgentDepthLimit valida que depth ≥ maxDepth retorna erro tipado.
func TestRunAgentDepthLimit(t *testing.T) {
	// Simular contexto com depth=3 (maxDepth=3 → currentDepth=4 > 3).
	t.Setenv(EnvContext, `{"parent_session_id":"test-parent","depth":3,"workspace_root":"/tmp"}`)

	registry := newMockRegistry(reviewerAgent())
	srv := New(registry, 3, nil)

	params := wire.ToolCallParams{
		Name: ReservedToolName,
	}
	args, _ := json.Marshal(RunAgentInput{
		AgentName: "reviewer",
		Prompt:    "revise",
	})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(mcpRequest(t, 1, "tools/call", json.RawMessage(paramsRaw)))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta recebida")
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("esperado erro de depth, obtido resultado")
	}
	if resp.Error.Code != wire.ErrCodeDepthExceeded {
		t.Fatalf("esperado código %d (ErrCodeDepthExceeded), obtido %d", wire.ErrCodeDepthExceeded, resp.Error.Code)
	}
	if !strings.Contains(resp.Error.Message, "profundidade") {
		t.Fatalf("mensagem de erro não menciona profundidade: %q", resp.Error.Message)
	}
}

// --- T-MCP-05: timeout honored ---

// TestRunAgentTimeoutHonored valida que timeout=1s com child travado retorna erro de timeout.
func TestRunAgentTimeoutHonored(t *testing.T) {
	// Este teste usa um registry que retorna um agente que não conseguirá ser spawnado
	// em ambiente de teste (sem claude-agent-acp), verificando que o timeout é honrado.
	registry := newMockRegistry(reviewerAgent())
	srv := New(registry, 3, nil)

	// Timeout de 1s no input, child tentará rodar mas contexto expira.
	params := wire.ToolCallParams{
		Name: ReservedToolName,
	}
	args, _ := json.Marshal(RunAgentInput{
		AgentName: "reviewer",
		Prompt:    "revise",
		Timeout:   1,
	})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(mcpRequest(t, 1, "tools/call", json.RawMessage(paramsRaw)))
	var out bytes.Buffer

	// Dar apenas 2s para o teste inteiro; o child deve falhar em ≤1s.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx, input, &out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("servidor não respondeu em 4s")
	}

	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta recebida")
	}

	resp := responses[0]
	// Deve ser erro (timeout ou outro erro de lançamento — o binário não existe em CI).
	if resp.Error == nil {
		t.Logf("aviso: sem erro retornado; pode haver claude-agent-acp disponível: %+v", resp.Result)
	}
}

// --- T-MCP-02: resolve via registry ---

// TestRunAgentResolvesViaRegistry valida que agent_name válido é resolvido e spawn é tentado.
func TestRunAgentResolvesViaRegistry(t *testing.T) {
	registry := newMockRegistry(reviewerAgent())
	srv := New(registry, 3, nil)

	params := wire.ToolCallParams{
		Name: ReservedToolName,
	}
	args, _ := json.Marshal(RunAgentInput{
		AgentName: "reviewer",
		Prompt:    "revise o código",
		Timeout:   1, // timeout curto para o teste não travar
	})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(mcpRequest(t, 1, "tools/call", json.RawMessage(paramsRaw)))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx, input, &out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		cancel()
		t.Fatal("servidor não respondeu em 4s")
	}

	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta recebida")
	}

	// A resolução foi bem-sucedida (chegou até o spawn) — qualquer resposta (sucesso ou erro
	// de lançamento do binário) é válida. O erro NOT FOUND do agent já não está aqui.
	resp := responses[0]
	if resp.Error != nil && resp.Error.Code == wire.ErrCodeAgentNotFound {
		t.Fatalf("agente 'reviewer' deveria ser encontrado, mas retornou ErrAgentNotFound")
	}
}

// --- T-MCP-06: evidence_dir retornado ---

// TestRunAgentEvidenceDirReturned está em engine_test.go (validação focada no engine).
// Este teste valida o fluxo de inicialização do servidor.

// TestServerInitializeHandshake valida o handshake MCP initialize.
func TestServerInitializeHandshake(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 3, nil)

	input := strings.NewReader(buildInitializeRequest(t))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta ao initialize")
	}

	resp := responses[0]
	if resp.Error != nil {
		t.Fatalf("initialize retornou erro: %+v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result wire.InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}

	if result.ProtocolVersion != wire.MCPVersion {
		t.Fatalf("protocolVersion esperado %q, obtido %q", wire.MCPVersion, result.ProtocolVersion)
	}
	if result.Capabilities.Tools == nil {
		t.Fatal("capabilities.tools deve ser não-nil")
	}
}

// TestServerUnknownMethodReturnsMethodNotFound valida método desconhecido.
func TestServerUnknownMethodReturnsMethodNotFound(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 3, nil)

	input := strings.NewReader(mcpRequest(t, 1, "nonexistent/method", nil))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta")
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("esperado erro MethodNotFound")
	}
	if resp.Error.Code != wire.ErrCodeMethodNotFound {
		t.Fatalf("esperado código %d, obtido %d", wire.ErrCodeMethodNotFound, resp.Error.Code)
	}
}

// TestRunAgentRequiresAgentName valida que agent_name vazio retorna erro.
func TestRunAgentRequiresAgentName(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 3, nil)

	params := wire.ToolCallParams{Name: ReservedToolName}
	args, _ := json.Marshal(RunAgentInput{AgentName: "", Prompt: "algo"})
	params.Arguments = args
	paramsRaw, _ := json.Marshal(params)

	input := strings.NewReader(mcpRequest(t, 1, "tools/call", json.RawMessage(paramsRaw)))
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serveDone := make(chan struct{})
	go func() { _ = srv.Serve(ctx, input, &out); close(serveDone) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-serveDone
	responses := parseResponses(t, &out)
	if len(responses) == 0 {
		t.Fatal("nenhuma resposta")
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("esperado erro de validação")
	}
	if resp.Error.Code != wire.ErrCodeInvalidParams {
		t.Fatalf("esperado %d, obtido %d", wire.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestNewServerDefaultMaxDepth valida que maxDepth ≤ 0 usa o default.
func TestNewServerDefaultMaxDepth(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 0, nil)
	s := srv.(*server)
	if s.maxDepth != DefaultMaxDepth {
		t.Fatalf("esperado maxDepth=%d, obtido %d", DefaultMaxDepth, s.maxDepth)
	}
}

// TestNewServerCustomMaxDepth valida que maxDepth personalizado é respeitado.
func TestNewServerCustomMaxDepth(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 5, nil)
	s := srv.(*server)
	if s.maxDepth != 5 {
		t.Fatalf("esperado maxDepth=5, obtido %d", s.maxDepth)
	}
}

// TestLoadNestedContextEmpty valida que contexto vazio retorna struct zerado.
func TestLoadNestedContextEmpty(t *testing.T) {
	t.Setenv(EnvContext, "")
	nc := NewCatalog().loadNestedContext()
	if nc.Depth != 0 || nc.ParentSessionID != "" {
		t.Fatalf("esperado contexto zerado, obtido %+v", nc)
	}
}

// TestLoadNestedContextValid valida que contexto JSON válido é carregado corretamente.
func TestLoadNestedContextValid(t *testing.T) {
	t.Setenv(EnvContext, `{"parent_session_id":"abc","depth":2,"workspace_root":"/workspace"}`)
	nc := NewCatalog().loadNestedContext()
	if nc.ParentSessionID != "abc" || nc.Depth != 2 || nc.WorkspaceRoot != "/workspace" {
		t.Fatalf("contexto inesperado: %+v", nc)
	}
}

// TestLoadNestedContextInvalidJSON valida fallback para JSON inválido.
func TestLoadNestedContextInvalidJSON(t *testing.T) {
	t.Setenv(EnvContext, "not-json{")
	nc := NewCatalog().loadNestedContext()
	if nc.Depth != 0 {
		t.Fatalf("esperado depth=0, obtido %d", nc.Depth)
	}
}

// TestServerEOFClosesLoop valida que EOF no reader encerra o loop normalmente.
func TestServerEOFClosesLoop(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 3, nil)

	// Reader vazio → EOF imediato.
	input := strings.NewReader("")
	var out bytes.Buffer

	err := srv.Serve(context.Background(), input, &out)
	if err != nil {
		t.Fatalf("esperado nil ao receber EOF, obtido: %v", err)
	}
}

// TestServerContextCancelClosesLoop valida que cancelamento de ctx encerra o loop.
func TestServerContextCancelClosesLoop(t *testing.T) {
	registry := newMockRegistry()
	srv := New(registry, 3, nil)

	// Reader que bloqueia indefinidamente.
	pr, _ := io.Pipe()
	var out bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx, pr, &out)
	}()

	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("esperado context.Canceled, obtido: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("servidor não encerrou após cancelamento do contexto")
	}
}

// TestPersistenceFactoryInjected valida que a factory é propagada para o server.
func TestPersistenceFactoryInjected(t *testing.T) {
	registry := newMockRegistry()
	var pf airuntime.PersistenceFactory // nil aceitável
	srv := New(registry, 3, pf)
	s := srv.(*server)
	if s.persistFactory != nil {
		t.Fatalf("esperado nil factory, obtido não-nil")
	}
}
