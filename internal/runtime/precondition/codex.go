package precondition

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var ErrCodexRPCUnavailable = fmt.Errorf("codex app-server RPC not available")

type CodexHookStatus struct {
	EventName   string
	Source      string
	TrustStatus string
}

type CodexHooksListResult struct {
	Hooks []CodexHookStatus
}

type CodexRPCClient interface {
	HooksList(ctx context.Context) (CodexHooksListResult, error)
}

type CodexAppServerClient struct {
	Binary string
}

func NewCodexAppServerClient(binary string) CodexAppServerClient {
	if binary == "" {
		binary = "codex"
	}
	return CodexAppServerClient{Binary: binary}
}

type codexRPCEnvelope struct {
	ID     *int   `json:"id"`
	Method string `json:"method"`
}

func (c CodexAppServerClient) HooksList(ctx context.Context) (CodexHooksListResult, error) {
	cmd := exec.CommandContext(ctx, c.Binary, "app-server")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return CodexHooksListResult{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CodexHooksListResult{}, err
	}

	if err := cmd.Start(); err != nil {
		return CodexHooksListResult{}, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)

	if err := sendCodexRPCRequest(stdin, 0, "initialize", map[string]any{
		"clientInfo": map[string]any{"name": "ai-spec-harness", "version": "0"},
	}); err != nil {
		return CodexHooksListResult{}, err
	}
	if _, err := readCodexRPCResponse(reader, 0); err != nil {
		return CodexHooksListResult{}, fmt.Errorf("codex app-server initialize: %w", err)
	}

	if err := sendCodexRPCRequest(stdin, 1, "hooks/list", map[string]any{}); err != nil {
		return CodexHooksListResult{}, err
	}
	body, err := readCodexRPCResponse(reader, 1)
	if err != nil {
		return CodexHooksListResult{}, fmt.Errorf("codex app-server hooks/list: %w", err)
	}

	var response struct {
		Result struct {
			Data []struct {
				Hooks []struct {
					EventName   string `json:"eventName"`
					Source      string `json:"source"`
					TrustStatus string `json:"trustStatus"`
				} `json:"hooks"`
			} `json:"data"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return CodexHooksListResult{}, err
	}
	if response.Error != nil {
		return CodexHooksListResult{}, fmt.Errorf("codex app-server hooks/list: %s", response.Error.Message)
	}

	out := CodexHooksListResult{}
	for _, group := range response.Result.Data {
		for _, h := range group.Hooks {
			out.Hooks = append(out.Hooks, CodexHookStatus{
				EventName:   h.EventName,
				Source:      h.Source,
				TrustStatus: h.TrustStatus,
			})
		}
	}
	return out, nil
}

func sendCodexRPCRequest(w interface{ Write([]byte) (int, error) }, id int, method string, params map[string]any) error {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = w.Write(payload)
	return err
}

func readCodexRPCResponse(reader *bufio.Reader, wantID int) ([]byte, error) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var envelope codexRPCEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}
		if envelope.ID == nil || *envelope.ID != wantID {
			continue
		}
		return line, nil
	}
}

func EvaluateCodexTrustedHash(ctx context.Context, client CodexRPCClient, timeout time.Duration) (specs.PreconditionState, error) {
	if client == nil {
		return specs.PreconditionUnknown, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := client.HooksList(callCtx)
	if err != nil {
		return specs.PreconditionUnknown, err
	}

	for _, hook := range result.Hooks {
		if hook.Source == "project" && hook.TrustStatus == "trusted" {
			return specs.PreconditionCurrent, nil
		}
	}
	return specs.PreconditionInert, nil
}
