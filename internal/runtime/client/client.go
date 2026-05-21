// Package client implementa o wrapper sobre o coder/acp-go-sdk para comunicação
// com o subprocesso claude-agent-acp via stdio JSON-RPC (ACP).
//
// Este pacote é o único ponto de importação do coder/acp-go-sdk no harness —
// além de internal/runtime/events/convert.go, que o usa como tipo de entrada.
// Ver ADR-009 e techspec §"Pontos de Integração".
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// Client é a interface para comunicação com um agente ACP.
// Define o contrato que o ACPRunner consome (techspec §"Interfaces Chave").
type Client interface {
	// Open abre uma sessão ACP com o subprocesso descrito pelo launcher,
	// enviando prompt como mensagem inicial.
	Open(ctx context.Context, launcher specs.Launcher, prompt string) error

	// Updates retorna o canal de eventos. Fechado em session_end ou erro.
	Updates() <-chan events.Event

	// Err retorna o erro acumulado após o fechamento do canal.
	Err() error

	// Close encerra a sessão e o subprocesso graciosamente.
	// Idempotente; seguro chamar múltiplas vezes.
	Close() error
}

// ClientFactory cria instâncias de Client.
// Injetada no ACPRunner para permitir substituição em testes.
type ClientFactory interface {
	New(workDir string) Client
}

// defaultClientFactory é a implementação de produção de ClientFactory.
type defaultClientFactory struct{}

// NewDefaultClientFactory cria a factory de produção.
func NewDefaultClientFactory() ClientFactory {
	return &defaultClientFactory{}
}

func (f *defaultClientFactory) New(workDir string) Client {
	return newACPClient(workDir, nil)
}

// IOProvider permite injetar io.ReadWriter customizados para testes in-process.
// Em produção, é nil e o client usa os pipes do subprocess.
type IOProvider interface {
	Provide() (io.Writer, io.Reader, error)
}

// acpClient é a implementação real de Client sobre coder/acp-go-sdk.
type acpClient struct {
	workDir    string
	ioProvider IOProvider // nil em produção; injeta pipeconn em testes

	mu      sync.Mutex
	cmd     *exec.Cmd
	eventCh chan events.Event
	err     error
	closed  atomic.Bool
	// sendMu serializa sends a eventCh com o close do canal em closeChannel.
	// SessionUpdate adquire RLock; closeChannel adquire Lock exclusivo.
	sendMu sync.RWMutex
	// permissionRequested marca que o agente pediu permissão durante o prompt turn.
	permissionRequested atomic.Bool
	once                sync.Once
	cancel              context.CancelFunc
}

// ErrPermissionDenied indica que o agente ACP solicitou permissão e o cliente
// cancelou imediatamente o prompt turn conforme RF-16.
var ErrPermissionDenied = errors.New("permission denied")

// newACPClient cria um novo acpClient sem ainda abrir a sessão.
// ioProvider pode ser nil (modo produção: spawn subprocess) ou um IOProvider (modo teste).
func newACPClient(workDir string, ioProvider IOProvider) *acpClient {
	return &acpClient{
		workDir:    workDir,
		ioProvider: ioProvider,
		eventCh:    make(chan events.Event, 64),
	}
}

// Open abre a sessão ACP. Em produção, faz spawn do subprocess.
// Em testes com IOProvider, usa os pipes injetados diretamente.
func (c *acpClient) Open(ctx context.Context, launcher specs.Launcher, prompt string) error {
	var stdinPipe io.Writer
	var stdoutPipe io.Reader
	var err error

	if c.ioProvider != nil {
		// Modo teste: IOProvider fornece os pipes diretamente.
		stdinPipe, stdoutPipe, err = c.ioProvider.Provide()
		if err != nil {
			return fmt.Errorf("ioProvider: %w", err)
		}
	} else {
		// Modo produção: spawn subprocess.
		stdinPipe, stdoutPipe, err = c.startProcess(ctx, launcher)
		if err != nil {
			return err
		}
	}

	conn, sessID, err := c.negotiate(ctx, stdinPipe, stdoutPipe, prompt)
	if err != nil {
		if c.cmd != nil {
			_ = c.killProcess()
		}
		return err
	}

	promptCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	// Inicia goroutine de leitura de updates (Prompt do SDK é bloqueante).
	go c.readLoop(promptCtx, conn, sessID, prompt)

	return nil
}

// startProcess faz spawn do subprocess ACP e retorna os pipes de stdin/stdout.
func (c *acpClient) startProcess(ctx context.Context, launcher specs.Launcher) (io.Writer, io.Reader, error) {
	cmd, args := launcher.Command()

	// R-SEC-001: exec.Command nunca usa shell — argumentos passados diretamente.
	//nolint:gosec // cmd e args vêm de specs.Launcher, não de input do usuário.
	c.cmd = exec.CommandContext(ctx, cmd, args...)
	if c.workDir != "" {
		c.cmd.Dir = c.workDir
	}
	c.cmd.Stderr = os.Stderr

	stdinPipe, err := c.cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("criar stdin pipe: %w", err)
	}
	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("criar stdout pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("iniciar subprocesso ACP: %w", err)
	}

	return stdinPipe, stdoutPipe, nil
}

// negotiate realiza o handshake ACP e cria a sessão.
func (c *acpClient) negotiate(ctx context.Context, w io.Writer, r io.Reader, _ string) (*acp.ClientSideConnection, acp.SessionId, error) {
	impl := &clientImpl{
		trySend: c.trySend,
		onPermission: func() {
			c.permissionRequested.Store(true)
			c.mu.Lock()
			cancel := c.cancel
			c.mu.Unlock()
			if cancel != nil {
				cancel()
			}
		},
	}
	conn := acp.NewClientSideConnection(impl, w, r)

	_, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapabilities{
				ReadTextFile:  false,
				WriteTextFile: false,
			},
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("inicializar protocolo ACP: %w", err)
	}

	sessResp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        c.workDir,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		return nil, "", fmt.Errorf("criar sessão ACP: %w", err)
	}

	return conn, sessResp.SessionId, nil
}

// readLoop envia o prompt e consome até o fim da sessão.
func (c *acpClient) readLoop(ctx context.Context, conn *acp.ClientSideConnection, sessID acp.SessionId, prompt string) {
	defer func() {
		if c.permissionRequested.Load() {
			c.closeChannel(ErrPermissionDenied)
			return
		}
		c.closeChannel(nil)
	}()

	resp, err := conn.Prompt(ctx, acp.PromptRequest{
		SessionId: sessID,
		Prompt:    []acp.ContentBlock{acp.TextBlock(prompt)},
	})
	if err != nil {
		if c.permissionRequested.Load() {
			c.closeChannel(ErrPermissionDenied)
			return
		}
		if ctx.Err() != nil {
			c.closeChannel(fmt.Errorf("sessão ACP cancelada: %w", ctx.Err()))
			return
		}
		c.closeChannel(fmt.Errorf("prompt ACP: %w", err))
		return
	}

	if evt, evtErr := sessionEndEventFromPromptResponse(resp); evtErr == nil {
		select {
		case c.eventCh <- evt:
		case <-ctx.Done():
		}
	}
}

func sessionEndEventFromPromptResponse(resp acp.PromptResponse) (events.Event, error) {
	raw, err := json.Marshal(map[string]any{
		"sessionUpdate": "session_end",
		"exitCode":      0,
		"reason":        string(resp.StopReason),
		"stopReason":    string(resp.StopReason),
	})
	if err != nil {
		return events.Event{}, err
	}
	return events.NewSessionEnd(time.Now().UTC(), 0, string(resp.StopReason), raw)
}

// closeChannel fecha o canal de eventos com um erro opcional. Idempotente.
// Adquire sendMu em modo exclusivo para que nenhum SessionUpdate concurrent
// possa fazer send enquanto o canal está sendo fechado (elimina a data race).
func (c *acpClient) closeChannel(err error) {
	c.once.Do(func() {
		c.mu.Lock()
		if err != nil {
			c.err = err
		}
		c.mu.Unlock()
		c.sendMu.Lock()
		c.closed.Store(true)
		close(c.eventCh)
		c.sendMu.Unlock()
	})
}

// trySend envia evt para eventCh de forma race-free. Retorna false se o canal
// já foi fechado ou está cheio. SessionUpdate usa esta função em vez de acessar
// eventCh diretamente.
func (c *acpClient) trySend(evt events.Event) {
	c.sendMu.RLock()
	defer c.sendMu.RUnlock()
	if c.closed.Load() {
		return
	}
	select {
	case c.eventCh <- evt:
	default:
	}
}

// Updates retorna o canal de eventos. Fechado em session_end ou erro.
func (c *acpClient) Updates() <-chan events.Event {
	return c.eventCh
}

// Err retorna o erro acumulado pós-fechamento do canal.
func (c *acpClient) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close encerra a sessão e o subprocesso. Idempotente.
func (c *acpClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	return c.killProcess()
}

// killProcess envia SIGTERM ao subprocesso, aguarda 2s e envia SIGKILL se necessário.
func (c *acpClient) killProcess() error {
	c.mu.Lock()
	cmd := c.cmd
	c.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return nil
	}
}

// clientImpl implementa acp.Client para receber SessionUpdate e RequestPermission.
type clientImpl struct {
	trySend      func(events.Event) // race-free send via acpClient.trySend
	onPermission func()
}

func (ci *clientImpl) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	evt, _ := events.FromACPUpdate("", n.Update)
	ci.trySend(evt)
	return nil
}

// RequestPermission cancela imediatamente (RF-16).
func (ci *clientImpl) RequestPermission(_ context.Context, _ acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	if ci.onPermission != nil {
		ci.onPermission()
	}
	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Cancelled: &acp.RequestPermissionOutcomeCancelled{},
		},
	}, nil
}

func (ci *clientImpl) ReadTextFile(_ context.Context, p acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, fmt.Errorf("ReadTextFile não suportado: %s", p.Path)
}

func (ci *clientImpl) WriteTextFile(_ context.Context, p acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, fmt.Errorf("WriteTextFile não suportado: %s", p.Path)
}

func (ci *clientImpl) CreateTerminal(_ context.Context, _ acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, fmt.Errorf("CreateTerminal não suportado")
}

func (ci *clientImpl) KillTerminal(_ context.Context, _ acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, fmt.Errorf("KillTerminal não suportado")
}

func (ci *clientImpl) ReleaseTerminal(_ context.Context, _ acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("ReleaseTerminal não suportado")
}

func (ci *clientImpl) TerminalOutput(_ context.Context, _ acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, fmt.Errorf("TerminalOutput não suportado")
}

func (ci *clientImpl) WaitForTerminalExit(_ context.Context, _ acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("WaitForTerminalExit não suportado")
}
