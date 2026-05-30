// Package mcpserver implementa o servidor MCP stdio interno do harness.
// Expõe uma única tool reservada run_agent, conforme ADR-014 F2-Claude.
// Decisão SDK: github.com/modelcontextprotocol/go-sdk estava instável em 2026-Q2;
// adotamos wire minimal em wire/ (~150 LoC) seguindo schema oficial MCP 2024-11-05.
// Ver execution_report.md §"Decisão MCP-SDK vs Wire Minimal".
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/mcpserver/wire"
)

// ReservedToolName é o único nome de tool exposto por este servidor MCP.
const ReservedToolName = "run_agent"

// DefaultTimeout é o timeout padrão para execução de um agente nested (segundos).
const DefaultTimeout = 300

// MaxTimeout é o timeout máximo permitido para execução de um agente nested (segundos).
const MaxTimeout = 1800

// DefaultMaxDepth é a profundidade máxima de aninhamento de agentes.
const DefaultMaxDepth = 3

// EnvContext é a variável de ambiente que carrega o contexto de execução nested.
const EnvContext = "AISPEC_RUN_AGENT_CONTEXT"

// EnvMaxDepth é a variável de ambiente para configurar a profundidade máxima.
const EnvMaxDepth = "AISPEC_MAX_AGENT_DEPTH"

// RunAgentInput descreve os parâmetros da tool run_agent.
type RunAgentInput struct {
	AgentName string `json:"agent_name"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model,omitempty"`
	Timeout   int    `json:"timeout,omitempty"` // segundos; default 300; max 1800
}

// RunAgentOutput descreve o resultado da tool run_agent.
type RunAgentOutput struct {
	Summary     string `json:"summary"`
	EvidenceDir string `json:"evidence_dir"`
}

// NestedExecutionContext é o contexto de execução propagado via env var.
type NestedExecutionContext struct {
	ParentSessionID string `json:"parent_session_id"`
	Depth           int    `json:"depth"`
	WorkspaceRoot   string `json:"workspace_root"`
}

// Server expõe a tool MCP run_agent via protocolo stdio.
type Server interface {
	// Serve roda o stdio MCP loop até o context cancelar ou EOF no reader.
	Serve(ctx context.Context, in io.Reader, out io.Writer) error
}

// server é a implementação interna de Server.
type server struct {
	registry       agents.Registry
	maxDepth       int
	persistFactory airuntime.PersistenceFactory
}

var _ Server = (*server)(nil)

// New cria um Server MCP configurado.
// registry: resolve agent_name para ResolvedAgent.
// maxDepth: profundidade máxima de aninhamento (default 3 se ≤ 0).
// persistFactory: fábrica de Persistence para child sessions; pode ser nil (sem persistência).
func New(registry agents.Registry, maxDepth int, persistFactory airuntime.PersistenceFactory) Server {
	if maxDepth <= 0 {
		maxDepth = NewCatalog().resolveMaxDepth()
	}
	return &server{
		registry:       registry,
		maxDepth:       maxDepth,
		persistFactory: persistFactory,
	}
}

// resolveMaxDepth resolve a profundidade máxima via env AISPEC_MAX_AGENT_DEPTH ou default.
func (c *Catalog) resolveMaxDepth() int {
	if v := os.Getenv(EnvMaxDepth); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxDepth
}

// Serve executa o loop stdio MCP até context cancelar ou EOF.
func (s *server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	codec := wire.NewCodec(in, out)

	// Derivar contexto de execução atual (depth do parent).
	parentCtx := NewCatalog().loadNestedContext()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := codec.ReadRequest()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			// Erro de parse — responder e continuar.
			_ = codec.WriteError(nil, wire.ErrCodeParseError, err.Error())
			continue
		}

		if handleErr := s.handleRequest(ctx, codec, req, parentCtx); handleErr != nil {
			// Erro fatal de escrita — encerrar o loop.
			return fmt.Errorf("mcpserver: escrever resposta: %w", handleErr)
		}
	}
}

// handleRequest despacha uma requisição JSON-RPC para o handler correto.
func (s *server) handleRequest(
	ctx context.Context,
	codec *wire.Codec,
	req wire.Request,
	parentCtx NestedExecutionContext,
) error {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(codec, req)
	case "notifications/initialized":
		// Notificação: sem resposta.
		return nil
	case "tools/list":
		return s.handleToolsList(codec, req)
	case "tools/call":
		return s.handleToolsCall(ctx, codec, req, parentCtx)
	default:
		return codec.WriteError(req.ID, wire.ErrCodeMethodNotFound,
			fmt.Sprintf("método não suportado: %s", req.Method))
	}
}

// handleInitialize responde ao handshake MCP.
func (s *server) handleInitialize(codec *wire.Codec, req wire.Request) error {
	result := wire.InitializeResult{
		ProtocolVersion: wire.MCPVersion,
		Capabilities: wire.Capabilities{
			Tools: &wire.ToolsCapability{ListChanged: false},
		},
		ServerInfo: wire.ServerInfo{
			Name:    "ai-spec-harness-mcpserver",
			Version: "1.0.0",
		},
	}
	return codec.WriteResult(req.ID, result)
}

// handleToolsList retorna a lista de tools (apenas run_agent).
func (s *server) handleToolsList(codec *wire.Codec, req wire.Request) error {
	tools := wire.ToolsListResult{
		Tools: []wire.ToolDefinition{
			{
				Name:        ReservedToolName,
				Description: "Executa um agente nested registrado no harness. Profundidade máxima: " + strconv.Itoa(s.maxDepth) + ".",
				InputSchema: NewCatalog().runAgentInputSchema(),
			},
		},
	}
	return codec.WriteResult(req.ID, tools)
}

// handleToolsCall despacha a execução da tool run_agent.
func (s *server) handleToolsCall(
	ctx context.Context,
	codec *wire.Codec,
	req wire.Request,
	parentCtx NestedExecutionContext,
) error {
	var params wire.ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return codec.WriteError(req.ID, wire.ErrCodeInvalidParams,
			fmt.Sprintf("parâmetros inválidos: %s", err))
	}

	if params.Name != ReservedToolName {
		return codec.WriteError(req.ID, wire.ErrCodeMethodNotFound,
			fmt.Sprintf("tool desconhecida: %s — apenas %q é suportada", params.Name, ReservedToolName))
	}

	var input RunAgentInput
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return codec.WriteError(req.ID, wire.ErrCodeInvalidParams,
				fmt.Sprintf("argumentos inválidos para run_agent: %s", err))
		}
	}

	// Validar agent_name obrigatório.
	if input.AgentName == "" {
		return codec.WriteError(req.ID, wire.ErrCodeInvalidParams, "agent_name é obrigatório")
	}

	// Verificar profundidade máxima.
	currentDepth := parentCtx.Depth + 1
	if currentDepth > s.maxDepth {
		return codec.WriteError(req.ID, wire.ErrCodeDepthExceeded,
			fmt.Sprintf("profundidade de aninhamento %d excede máximo %d", currentDepth, s.maxDepth))
	}

	// Resolver agente no registry.
	agent, err := s.registry.Resolve(input.AgentName)
	if err != nil {
		return codec.WriteError(req.ID, wire.ErrCodeAgentNotFound,
			fmt.Sprintf("agente não encontrado: %s", err))
	}

	// Resolver timeout.
	timeoutSec := input.Timeout
	if timeoutSec <= 0 {
		timeoutSec = DefaultTimeout
	}
	if timeoutSec > MaxTimeout {
		timeoutSec = MaxTimeout
	}

	// Criar contexto com timeout para a child session.
	childCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Construir contexto nested para o child.
	childNestedCtx := NestedExecutionContext{
		ParentSessionID: parentCtx.ParentSessionID,
		Depth:           currentDepth,
		WorkspaceRoot:   parentCtx.WorkspaceRoot,
	}

	// Spawnar child session via engine.
	output, err := NewCatalog().spawnNestedSession(childCtx, SpawnParams{
		Agent:          agent,
		Input:          input,
		NestedCtx:      childNestedCtx,
		ParentCtx:      parentCtx,
		PersistFactory: s.persistFactory,
	})
	if err != nil {
		errCode := wire.ErrCodeInternal
		if childCtx.Err() == context.DeadlineExceeded {
			errCode = wire.ErrCodeTimeout
			err = fmt.Errorf("timeout após %ds: %w", timeoutSec, err)
		}
		return codec.WriteError(req.ID, errCode, err.Error())
	}

	// Serializar output como texto para o conteúdo da tool.
	outputJSON, _ := json.Marshal(output)
	result := wire.ToolCallResult{
		Content: []wire.ToolCallContent{
			{Type: "text", Text: string(outputJSON)},
		},
	}
	return codec.WriteResult(req.ID, result)
}

// loadNestedContext carrega o NestedExecutionContext da env var AISPEC_RUN_AGENT_CONTEXT.
// Retorna contexto zerado (depth=0) quando a env var não está definida (sessão raiz).
func (c *Catalog) loadNestedContext() NestedExecutionContext {
	raw := os.Getenv(EnvContext)
	if raw == "" {
		return NestedExecutionContext{}
	}
	var nc NestedExecutionContext
	if err := json.Unmarshal([]byte(raw), &nc); err != nil {
		return NestedExecutionContext{}
	}
	return nc
}

// runAgentInputSchema retorna o JSON Schema da tool run_agent.
func (c *Catalog) runAgentInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_name": map[string]any{
				"type":        "string",
				"description": "Nome do agente declarativo registrado no harness.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Prompt a ser enviado ao agente.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Modelo a ser usado (opcional; usa default do agente).",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout em segundos (default 300; máximo 1800).",
				"minimum":     1,
				"maximum":     MaxTimeout,
			},
		},
		"required": []string{"agent_name", "prompt"},
	}
}
