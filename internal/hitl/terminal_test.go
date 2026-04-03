package hitl

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestTerminalPrompterApprove(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("A\n")
	output := &bytes.Buffer{}
	prompter := NewTerminalPrompter(input, output, platform.FakeEditor{})

	result, err := prompter.Prompt(context.Background(), "conteudo")
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != ActionApprove {
		t.Fatalf("action = %v", result.Action)
	}
	if !strings.Contains(output.String(), "[A] Aprovar") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestTerminalPrompterInvalidThenRedo(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("x\nR\n")
	output := &bytes.Buffer{}
	prompter := NewTerminalPrompter(input, output, platform.FakeEditor{})

	result, err := prompter.Prompt(context.Background(), "conteudo")
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != ActionRedo {
		t.Fatalf("action = %v", result.Action)
	}
	if !strings.Contains(output.String(), "Ação inválida") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestTerminalPrompterEdit(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("E\n")
	prompter := NewTerminalPrompter(input, &bytes.Buffer{}, platform.FakeEditor{
		EditFunc: func(_ context.Context, content string) (string, error) {
			return content + " editado", nil
		},
	})

	result, err := prompter.Prompt(context.Background(), "conteudo")
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != ActionEdit || result.Output != "conteudo editado" {
		t.Fatalf("result = %+v", result)
	}
}

func TestFakePrompter(t *testing.T) {
	t.Parallel()

	prompter := NewFakePrompter(
		PromptResult{Action: ActionApprove},
		PromptResult{Action: ActionExit},
	)

	result, err := prompter.Prompt(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != ActionApprove {
		t.Fatalf("action = %v", result.Action)
	}
}
