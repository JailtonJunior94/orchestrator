package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Editor opens content in an external editor and returns the edited content.
type Editor interface {
	Edit(ctx context.Context, content string) (string, error)
}

// ExternalEditor uses $EDITOR or a platform fallback editor.
type ExternalEditor struct {
	editorCommand string
}

// NewEditor creates a production editor adapter.
func NewEditor() Editor {
	return ExternalEditor{editorCommand: resolveEditorCommand()}
}

// NewEditorWithCommand creates an editor that always uses the provided binary.
func NewEditorWithCommand(command string) Editor {
	return ExternalEditor{editorCommand: command}
}

// Edit writes the content to a temp file, opens the editor, then returns the edited bytes.
func (e ExternalEditor) Edit(ctx context.Context, content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "orq-edit-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file for editor: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("writing editor temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("closing editor temp file: %w", err)
	}

	cmd := exec.CommandContext(ctx, e.editorCommand, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("opening editor %q: %w", e.editorCommand, err)
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("reading editor temp file: %w", err)
	}

	return string(edited), nil
}

func resolveEditorCommand() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if runtime.GOOS == "windows" {
		return "notepad"
	}

	return "vi"
}
