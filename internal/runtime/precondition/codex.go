package precondition

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var ErrCodexRPCUnavailable = fmt.Errorf("codex app-server RPC not available")

type CodexHookStatus struct {
	EventName   string
	Source      string
	TrustStatus string
	Key         string
	CurrentHash string
}

type CodexHooksListResult struct {
	Hooks []CodexHookStatus
}

type CodexRPCClient interface {
	HooksList(ctx context.Context) (CodexHooksListResult, error)
}

type CodexAppServerClient struct {
	Binary  string
	WorkDir string
}

func NewCodexAppServerClient(binary, workDir string) CodexAppServerClient {
	if binary == "" {
		binary = "codex"
	}
	return CodexAppServerClient{Binary: binary, WorkDir: workDir}
}

type codexRPCEnvelope struct {
	ID     *int   `json:"id"`
	Method string `json:"method"`
}

func (c CodexAppServerClient) HooksList(ctx context.Context) (CodexHooksListResult, error) {
	cmd := exec.CommandContext(ctx, c.Binary, "app-server")
	cmd.Dir = c.WorkDir

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
		"clientInfo":   map[string]any{"name": "ai-spec-harness", "version": "0"},
		"capabilities": map[string]any{"experimentalApi": true},
	}); err != nil {
		return CodexHooksListResult{}, err
	}
	if _, err := readCodexRPCResponse(reader, 0); err != nil {
		return CodexHooksListResult{}, fmt.Errorf("codex app-server initialize: %w", err)
	}

	params := map[string]any{}
	if c.WorkDir != "" {
		params["cwds"] = []string{c.WorkDir}
	}
	if err := sendCodexRPCRequest(stdin, 1, "hooks/list", params); err != nil {
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
					Key         string `json:"key"`
					CurrentHash string `json:"currentHash"`
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
				Key:         h.Key,
				CurrentHash: h.CurrentHash,
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

type CodexTrustPoint struct {
	EventName string
	State     specs.PreconditionState
}

type CodexTrustReport struct {
	Points []CodexTrustPoint
}

func (r CodexTrustReport) State() specs.PreconditionState {
	if len(r.Points) == 0 {
		return specs.PreconditionUnknown
	}
	for _, point := range r.Points {
		if point.State != specs.PreconditionCurrent {
			return point.State
		}
	}
	return specs.PreconditionCurrent
}

func EvaluateCodexTrustedHash(ctx context.Context, client CodexRPCClient, timeout time.Duration, requiredEvents []string) (CodexTrustReport, error) {
	if client == nil {
		return CodexTrustReport{}, nil
	}
	if len(requiredEvents) == 0 {
		return CodexTrustReport{}, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := client.HooksList(callCtx)
	if err != nil {
		return CodexTrustReport{}, err
	}

	report := CodexTrustReport{Points: make([]CodexTrustPoint, 0, len(requiredEvents))}
	for _, event := range requiredEvents {
		report.Points = append(report.Points, CodexTrustPoint{
			EventName: event,
			State:     evaluateCodexEventTrust(result.Hooks, event),
		})
	}
	return report, nil
}

func evaluateCodexEventTrust(hooks []CodexHookStatus, event string) specs.PreconditionState {
	found := false
	for _, hook := range hooks {
		if hook.Source != "project" || !strings.EqualFold(hook.EventName, event) {
			continue
		}
		found = true
		if hook.TrustStatus != "trusted" {
			return specs.PreconditionInert
		}
	}
	if !found {
		return specs.PreconditionInert
	}
	return specs.PreconditionCurrent
}
