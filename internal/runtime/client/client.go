// Package client implementa o wrapper sobre o coder/acp-go-sdk para comunicação
// com o subprocesso claude-agent-acp via stdio JSON-RPC (ACP).
//
// Este pacote é o único ponto de importação do coder/acp-go-sdk no harness —
// além de internal/runtime/events/convert.go, que o usa como tipo de entrada.
// Ver ADR-009 e techspec §"Pontos de Integração".
//
// Implementação completa entregue na task 8.0 (ACP Client + Fake Server + Renderer).
// Este arquivo existe na task 3.0 para pinar a dependência no go.mod
// e garantir que go mod tidy não a remova antes de convert.go ser criado na task 4.0.
package client

import (
	// Importado para pinar a dependência em go.mod.
	// A implementação real do acpClient usa este pacote extensivamente (task 8.0).
	_ "github.com/coder/acp-go-sdk"
)
