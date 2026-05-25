# Live Tests ACP — `tests/integration/acp_live`

Testes de integração live que executam o runtime ACP com o agente real `claude-agent-acp`.
Protegidos pela build tag `acp_live` — não compilados por `go test ./...` sem a tag.

## Pré-requisitos

Pelo menos um dos seguintes deve estar disponível:

- **Binário direto:** `claude-agent-acp` instalado no PATH
  ```bash
  npm install -g @agentclientprotocol/claude-agent-acp@0.37.0
  # ou via Homebrew se disponível
  ```
- **Via npx:** `npx` e `node` disponíveis no PATH (fallback automático)
  ```bash
  node --version  # requer Node.js 18+
  npx --version
  ```

**Autenticação** (opcional para handshake parcial):
- `ANTHROPIC_API_KEY` configurado no ambiente, **OU**
- Configuração local `~/.claude/` com sessão válida (após `claude` login)

Sem autenticação, o teste valida apenas o handshake inicial (até `permission_denied`) e passa.

## Executar os Testes

```bash
# Testes live (requer binário + opcional: auth)
go test -tags=acp_live ./tests/integration/acp_live

# Com verbose para ver detalhes do handshake
go test -tags=acp_live -v ./tests/integration/acp_live

# Via Makefile
make test-acp-live

# Com API key explícita
ANTHROPIC_API_KEY=sk-ant-... go test -tags=acp_live ./tests/integration/acp_live
```

## Validar que o Pacote NÃO Compila sem a Tag

```bash
go list -f '{{.GoFiles}}' ./tests/integration/acp_live
# Deve retornar: []
```

## Custo Esperado

- **Handshake sem auth:** 0 tokens (falha antes de enviar prompt)
- **Handshake completo (`echo OK`):** ~50–200 tokens dependendo do modelo
- **Timeout watchdog:** 30s de inatividade máxima por teste

## ADR de Referência

Ver `.specs/adr/009-acp-protocol-adoption.md` para decisões de arquitetura sobre
o protocolo ACP, versão pinada do `claude-agent-acp` e política de upgrade.
