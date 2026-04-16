# Compozy - Analise de Comunicacao com Ferramentas de IA

## 1. Visao Geral da Arquitetura

Compozy e uma CLI em Go que orquestra agentes de IA (Claude Code, Codex, Gemini, Copilot, etc.) atraves do **ACP (Agent Client Protocol)** - um protocolo JSON-RPC 2.0 sobre pipes stdin/stdout.

**Pacotes principais:**

| Pacote | Responsabilidade |
|---|---|
| `cmd/compozy/main.go` | Entrypoint |
| `internal/core/agent/` | Cliente ACP, gerenciamento de sessoes, registro de agentes |
| `internal/core/run/executor/` | Engine de execucao (paralelo/sequencial) |
| `internal/core/subprocess/` | Spawn de processos e transporte JSON-RPC |
| `internal/core/model/` | Constantes, tipos, runtime config |

## 2. Protocolo de Comunicacao: ACP sobre JSON-RPC 2.0

Compozy **NAO** invoca as CLIs diretamente para parsear texto de saida. Em vez disso, utiliza **binarios adaptadores ACP** que expoe um protocolo padronizado.

### Fluxo de Comunicacao

```
┌─────────┐    stdin (JSON-RPC)     ┌──────────────────┐
│ Compozy │ ──────────────────────► │ ACP Adapter      │
│  (host) │ ◄────────────────────── │ (claude-agent-acp│
│         │    stdout (JSON-RPC)    │  codex-acp, etc) │
└─────────┘                         └──────────────────┘
```

### Etapas do Protocolo

1. **Spawn do subprocesso** - o binario ACP do agente e lancado via `subprocess.Launch()`
2. **Transporte JSON-RPC** - mensagens JSON-RPC 2.0 delimitadas por linha sao trocadas via stdin/stdout
3. **Handshake ACP** - `conn.Initialize()` envia versao do protocolo e capabilities do cliente
4. **Criacao de sessao** - `conn.NewSession()` abre sessao com diretorio de trabalho e servidores MCP
5. **Execucao do prompt** - `conn.Prompt()` envia o prompt como `acp.TextBlock`
6. **Streaming de atualizacoes** - o agente envia `SessionUpdate` notifications (chunks de mensagem, tool calls, pensamentos)
7. **Operacoes de arquivo** - o agente pode solicitar `ReadTextFile`/`WriteTextFile` atraves do ACP
8. **Permissoes** - `RequestPermission` auto-aprova a primeira opcao (modo nao-interativo)

## 3. Configuracao por Ferramenta

### Claude Code

| Campo | Valor |
|---|---|
| **Binario** | `claude-agent-acp` |
| **Fallback** | `npx --yes @agentclientprotocol/claude-agent-acp` |
| **Modelo padrao** | `opus` |
| **SupportsAddDirs** | `true` |
| **FullAccessModeID** | `bypassPermissions` |
| **Args extras** | Nenhum |
| **Docs** | https://github.com/agentclientprotocol/claude-agent-acp |

**Comando efetivo:**
```bash
claude-agent-acp
```

### Codex (OpenAI)

| Campo | Valor |
|---|---|
| **Binario** | `codex-acp` |
| **Fallback** | `npx --yes @zed-industries/codex-acp` |
| **Modelo padrao** | `gpt-5.4` |
| **SupportsAddDirs** | `true` |
| **Docs** | https://github.com/zed-industries/codex-acp |

**Comando efetivo (full access):**
```bash
codex-acp -c approval_policy="never" -c sandbox_mode="danger-full-access" -c web_search="live"
```

### Gemini CLI

| Campo | Valor |
|---|---|
| **Binario** | `gemini --acp` |
| **Fallback** | `npx --yes @google/gemini-cli --acp` |
| **Probe** | `gemini --acp --help` |
| **Modelo padrao** | `gemini-2.5-pro` |
| **Args extras** | Nenhum |
| **Docs** | https://geminicli.com |

**Comando efetivo:**
```bash
gemini --acp
```

### Copilot CLI

| Campo | Valor |
|---|---|
| **Binario** | `copilot --acp` |
| **Fallback** | `npx --yes @github/copilot --acp` |
| **Modelo padrao** | `claude-sonnet-4.6` |
| **Docs** | https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server |

**Outros agentes suportados:** Droid, Cursor, OpenCode, Pi.

## 4. Envio do Prompt

O prompt NAO e passado como argumento de CLI. Ele e enviado pelo pipe JSON-RPC apos o estabelecimento da sessao:

```go
c.conn.Prompt(ctx, acp.PromptRequest{
    SessionId: sessionResp.SessionId,
    Prompt:    []acp.ContentBlock{acp.TextBlock(string(req.Prompt))},
})
```

O conteudo e encapsulado em `ContentBlock` do tipo texto, seguindo a especificacao ACP.

## 5. Captura de Output

O output e capturado via **streaming de notificacoes ACP**. O callback `SessionUpdate` recebe:

| Tipo de Update | Descricao |
|---|---|
| `AgentMessageChunk` | Texto de saida do agente |
| `AgentThoughtChunk` | Raciocinio/pensamento (chain-of-thought) |
| `ToolCall` | Eventos de uso de ferramenta (editar arquivo, executar comando) |
| `ToolCallUpdate` | Atualizacoes incrementais de tool calls |

Essas notificacoes sao convertidas de tipos do SDK ACP para tipos internos (`model.SessionUpdate`) e publicadas em um channel bufferizado (capacidade: 1024).

## 6. Modelo de Orquestracao

### Execucao de Jobs

- **Job** = uma sessao de agente IA (unidade de execucao)
- **Sequencial** quando modo `PRDTasks` ou `Concurrent == 1`
- **Paralelo** com semaforo limitando concorrencia (`cfg.Concurrent`)

### Retry e Resiliencia

- Retries configuraveis com backoff multiplicativo
- Timeout maximo de 30 minutos
- Hooks de extensao: `job.pre_execute`, `job.post_execute`, `job.pre_retry`

### Graceful Shutdown

1. Fecha stdin cooperativamente
2. Envia `SIGTERM`
3. Envia `SIGKILL` como ultimo recurso

## 7. Configuracao

| Mecanismo | Exemplo |
|---|---|
| Selecao do agente | `RuntimeConfig.IDE` = `"claude"`, `"codex"`, `"gemini"` |
| Override de modelo | `cfg.Model` |
| Modo de acesso | `cfg.AccessMode` = `"default"` ou `"full"` |
| Config do workspace | `.compozy/config.toml` |
| Servidores MCP | Anexados a sessoes |
| Variaveis de ambiente | Per-spec `EnvVars` + per-session `ExtraEnv` |

## 8. Diferenca Fundamental vs Abordagem por Subprocesso Direto

| Aspecto | Compozy (ACP) | Subprocesso direto |
|---|---|---|
| **Protocolo** | JSON-RPC 2.0 estruturado | Parse de stdout texto livre |
| **Streaming** | Chunks tipados (mensagem, tool call, pensamento) | Texto bruto |
| **Sessao** | Create/resume/load nativo | Nao existe |
| **Filesystem** | Agente solicita read/write via protocolo | Agente age diretamente |
| **Permissoes** | Controle granular via ACP | Flags de CLI |
| **Multi-agente** | Mesmo protocolo para todos | Adapter diferente por provider |

## 9. Dependencia Principal

O SDK utilizado e o `github.com/coder/acp-go-sdk`, que implementa:
- Transporte JSON-RPC sobre stdio
- Tipos ACP (ContentBlock, SessionUpdate, PromptRequest)
- Gerenciamento de conexao e sessao

## 10. Resumo

Compozy unifica a comunicacao com todas as ferramentas de IA atraves de um unico protocolo (ACP/JSON-RPC 2.0). Cada ferramenta possui um binario adaptador ACP dedicado que traduz o protocolo padronizado para a API/CLI especifica da ferramenta. Isso permite adicionar novos agentes sem alterar o core da engine - basta registrar o novo `Spec` com o binario correto no registry.
