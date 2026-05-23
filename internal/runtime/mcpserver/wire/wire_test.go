package wire

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func rawID(t *testing.T, id int) *json.RawMessage {
	t.Helper()
	b, _ := json.Marshal(id)
	m := json.RawMessage(b)
	return &m
}

// TestCodecReadRequestSuccess valida leitura de uma requisição JSON-RPC válida.
func TestCodecReadRequestSuccess(t *testing.T) {
	rawMsg := json.RawMessage(`1`)
	req := Request{
		JSONRPC: "2.0",
		ID:      &rawMsg,
		Method:  "tools/list",
	}
	b, _ := json.Marshal(req)
	in := strings.NewReader(string(b) + "\n")
	var out bytes.Buffer

	codec := NewCodec(in, &out)
	got, err := codec.ReadRequest()
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if got.Method != "tools/list" {
		t.Fatalf("esperado method=tools/list, obtido %q", got.Method)
	}
}

// TestCodecReadRequestEOF valida que EOF é retornado quando o reader está vazio.
func TestCodecReadRequestEOF(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer
	codec := NewCodec(in, &out)

	_, err := codec.ReadRequest()
	if err != io.EOF {
		t.Fatalf("esperado io.EOF, obtido %v", err)
	}
}

// TestCodecReadRequestInvalidJSON valida erro em JSON inválido.
func TestCodecReadRequestInvalidJSON(t *testing.T) {
	in := strings.NewReader("{invalid-json}\n")
	var out bytes.Buffer
	codec := NewCodec(in, &out)

	_, err := codec.ReadRequest()
	if err == nil {
		t.Fatal("esperado erro para JSON inválido")
	}
}

// TestCodecWriteResult valida escrita de uma resposta de sucesso.
func TestCodecWriteResult(t *testing.T) {
	var out bytes.Buffer
	codec := NewCodec(strings.NewReader(""), &out)

	err := codec.WriteResult(rawID(t, 42), map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("WriteResult: %v", err)
	}

	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("esperado sem erro, obtido %+v", resp.Error)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("esperado jsonrpc=2.0, obtido %q", resp.JSONRPC)
	}
}

// TestCodecWriteError valida escrita de uma resposta de erro.
func TestCodecWriteError(t *testing.T) {
	var out bytes.Buffer
	codec := NewCodec(strings.NewReader(""), &out)

	err := codec.WriteError(rawID(t, 1), ErrCodeMethodNotFound, "método não encontrado")
	if err != nil {
		t.Fatalf("WriteError: %v", err)
	}

	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("esperado erro na resposta")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("esperado código %d, obtido %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "método não encontrado" {
		t.Fatalf("mensagem inesperada: %q", resp.Error.Message)
	}
}

// TestCodecWriteErrorNilID valida escrita de erro sem ID.
func TestCodecWriteErrorNilID(t *testing.T) {
	var out bytes.Buffer
	codec := NewCodec(strings.NewReader(""), &out)

	err := codec.WriteError(nil, ErrCodeParseError, "parse error")
	if err != nil {
		t.Fatalf("WriteError nil id: %v", err)
	}

	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != nil {
		t.Fatal("esperado ID nil na resposta")
	}
}

// TestCodecWriteResponse valida escrita direta de Response.
func TestCodecWriteResponse(t *testing.T) {
	var out bytes.Buffer
	codec := NewCodec(strings.NewReader(""), &out)

	resp := Response{
		JSONRPC: "2.0",
		ID:      rawID(t, 99),
		Result:  "ok",
	}
	if err := codec.WriteResponse(resp); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}

	var got Response
	if err := json.NewDecoder(&out).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Fatalf("jsonrpc inesperado: %q", got.JSONRPC)
	}
}

// TestErrorCodes valida que os códigos de erro estão corretos.
func TestErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
	}{
		{"ParseError", ErrCodeParseError},
		{"InvalidRequest", ErrCodeInvalidRequest},
		{"MethodNotFound", ErrCodeMethodNotFound},
		{"InvalidParams", ErrCodeInvalidParams},
		{"Internal", ErrCodeInternal},
		{"AgentNotFound", ErrCodeAgentNotFound},
		{"DepthExceeded", ErrCodeDepthExceeded},
		{"Timeout", ErrCodeTimeout},
	}
	for _, tc := range cases {
		if tc.code == 0 {
			t.Errorf("%s: código não pode ser zero", tc.name)
		}
	}
}

// TestMCPVersion valida que a versão MCP está definida.
func TestMCPVersion(t *testing.T) {
	if MCPVersion == "" {
		t.Fatal("MCPVersion não pode ser vazio")
	}
}

// TestReadMultipleRequests valida leitura de múltiplas requisições do mesmo stream.
func TestReadMultipleRequests(t *testing.T) {
	rawMsg1 := json.RawMessage(`1`)
	rawMsg2 := json.RawMessage(`2`)
	req1 := Request{JSONRPC: "2.0", ID: &rawMsg1, Method: "initialize"}
	req2 := Request{JSONRPC: "2.0", ID: &rawMsg2, Method: "tools/list"}

	b1, _ := json.Marshal(req1)
	b2, _ := json.Marshal(req2)
	in := strings.NewReader(string(b1) + "\n" + string(b2) + "\n")

	codec := NewCodec(in, &bytes.Buffer{})

	got1, err := codec.ReadRequest()
	if err != nil {
		t.Fatalf("ReadRequest 1: %v", err)
	}
	if got1.Method != "initialize" {
		t.Fatalf("request 1 method inesperado: %q", got1.Method)
	}

	got2, err := codec.ReadRequest()
	if err != nil {
		t.Fatalf("ReadRequest 2: %v", err)
	}
	if got2.Method != "tools/list" {
		t.Fatalf("request 2 method inesperado: %q", got2.Method)
	}

	_, err = codec.ReadRequest()
	if err != io.EOF {
		t.Fatalf("esperado EOF após 2 requests, obtido %v", err)
	}
}
