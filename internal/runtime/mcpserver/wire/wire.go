// Package wire implementa o wire minimal do protocolo MCP (Model Context Protocol) stdio.
// Decisão de design: github.com/modelcontextprotocol/go-sdk estava indisponível ou instável
// em 2026-Q2; implementamos ~150 LoC de wire MCP seguindo o schema oficial JSON-RPC 2.0.
// Ver task-1.0 execution_report.md §"Decisão MCP-SDK vs Wire Minimal".
package wire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// Versão do protocolo MCP suportada por este wire.
const MCPVersion = "2024-11-05"

// --- Tipos JSON-RPC 2.0 ---

// Request representa uma requisição JSON-RPC 2.0.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response representa uma resposta JSON-RPC 2.0.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError representa um erro JSON-RPC 2.0.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Códigos de erro MCP/JSON-RPC.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603

	// Códigos específicos MCP (range application).
	ErrCodeAgentNotFound = -32001
	ErrCodeDepthExceeded = -32002
	ErrCodeTimeout       = -32003
)

// --- Tipos MCP ---

// ToolDefinition define uma tool MCP.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// InitializeResult é o resultado da chamada initialize.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities define as capacidades do servidor MCP.
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability indica suporte a tools.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo contém informações do servidor MCP.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListResult é o resultado de tools/list.
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolCallParams contém os parâmetros de tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult é o resultado de tools/call.
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

// ToolCallContent representa um bloco de conteúdo no resultado de uma tool.
type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// --- Codec stdio ---

// Codec lê e escreve mensagens JSON-RPC via stdin/stdout (newline-delimited JSON).
type Codec struct {
	scanner *bufio.Scanner
	encoder *json.Encoder
	out     io.Writer
}

// NewCodec cria um Codec para os readers/writers fornecidos.
func NewCodec(in io.Reader, out io.Writer) *Codec {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line
	return &Codec{
		scanner: sc,
		encoder: json.NewEncoder(out),
		out:     out,
	}
}

// ReadRequest lê a próxima requisição JSON-RPC do stream.
// Retorna io.EOF quando o stream encerrar.
func (c *Codec) ReadRequest() (Request, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return Request{}, fmt.Errorf("wire: leitura stdin: %w", err)
		}
		return Request{}, io.EOF
	}
	var req Request
	if err := json.Unmarshal(c.scanner.Bytes(), &req); err != nil {
		return Request{}, fmt.Errorf("wire: parse JSON-RPC: %w", err)
	}
	return req, nil
}

// WriteResponse escreve uma resposta JSON-RPC no stream.
func (c *Codec) WriteResponse(resp Response) error {
	if err := c.encoder.Encode(resp); err != nil {
		return fmt.Errorf("wire: escrever resposta: %w", err)
	}
	return nil
}

// WriteError escreve uma resposta de erro JSON-RPC.
func (c *Codec) WriteError(id *json.RawMessage, code int, msg string) error {
	return c.WriteResponse(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

// WriteResult escreve uma resposta de sucesso JSON-RPC.
func (c *Codec) WriteResult(id *json.RawMessage, result interface{}) error {
	return c.WriteResponse(Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}
