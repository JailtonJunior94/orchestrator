package hitl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// TerminalPrompter implements HITL via stdin/stdout.
type TerminalPrompter struct {
	reader io.Reader
	writer io.Writer
	editor platform.Editor
}

// NewTerminalPrompter creates a terminal-backed Prompter.
func NewTerminalPrompter(reader io.Reader, writer io.Writer, editor platform.Editor) *TerminalPrompter {
	return &TerminalPrompter{
		reader: reader,
		writer: writer,
		editor: editor,
	}
}

// Prompt renders output and waits for a valid action.
func (p *TerminalPrompter) Prompt(ctx context.Context, output string) (PromptResult, error) {
	if _, err := fmt.Fprintf(p.writer, "\n--- Output ---\n%s\n---------------\n\n[A] Aprovar  [E] Editar  [R] Refazer  [S] Sair\n> ", output); err != nil {
		return PromptResult{}, err
	}

	scanner := bufio.NewScanner(p.reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return PromptResult{}, ctx.Err()
		default:
		}

		switch strings.ToUpper(strings.TrimSpace(scanner.Text())) {
		case "A":
			return PromptResult{Action: ActionApprove}, nil
		case "E":
			edited, err := p.editor.Edit(ctx, output)
			if err != nil {
				return PromptResult{}, err
			}
			return PromptResult{Action: ActionEdit, Output: edited}, nil
		case "R":
			return PromptResult{Action: ActionRedo}, nil
		case "S":
			return PromptResult{Action: ActionExit}, nil
		default:
			if _, err := fmt.Fprint(p.writer, "Ação inválida. Use A, E, R ou S.\n> "); err != nil {
				return PromptResult{}, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return PromptResult{}, err
	}

	return PromptResult{}, io.EOF
}

// PromptPermission renders an ACP permission request and waits for an allow,
// deny or cancel decision.
func (p *TerminalPrompter) PromptPermission(ctx context.Context, request PermissionRequest) (PermissionResult, error) {
	if _, err := fmt.Fprintf(
		p.writer,
		"\n--- Permission Request ---\nProvider: %s\nWorkflow: %s\nTool: %s\nDetails: %s\n--------------------------\n\n[A] Allow  [D] Deny  [C] Cancel\n> ",
		request.Provider,
		request.Workflow,
		request.Title,
		request.Details,
	); err != nil {
		return PermissionResult{}, err
	}

	scanner := bufio.NewScanner(p.reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return PermissionResult{}, ctx.Err()
		default:
		}

		switch strings.ToUpper(strings.TrimSpace(scanner.Text())) {
		case "A":
			return PermissionResult{Decision: PermissionAllow}, nil
		case "D":
			return PermissionResult{Decision: PermissionDeny}, nil
		case "C":
			return PermissionResult{Decision: PermissionCancel}, nil
		default:
			if _, err := fmt.Fprint(p.writer, "Ação inválida. Use A, D ou C.\n> "); err != nil {
				return PermissionResult{}, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return PermissionResult{}, err
	}

	return PermissionResult{}, io.EOF
}
