package acp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/jailtonjunior/orchestrator/internal/acp"
)

func TestFSHandler_ReadTextFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "hello world"
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := acp.NewFSHandler(dir, nil)
	resp, err := handler.ReadTextFile(context.Background(), acpsdk.ReadTextFileRequest{Path: "test.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != content {
		t.Fatalf("content = %q, want %q", resp.Content, content)
	}
}

func TestFSHandler_ReadTextFile_PathTraversal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	_, err := handler.ReadTextFile(context.Background(), acpsdk.ReadTextFileRequest{Path: "../../etc/passwd"})
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !isPathOutsideWorkDir(err) {
		t.Fatalf("expected ErrPathOutsideWorkDir, got %v", err)
	}
}

func TestFSHandler_ReadTextFile_AbsoluteOutsideCWD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	_, err := handler.ReadTextFile(context.Background(), acpsdk.ReadTextFileRequest{Path: "/etc/passwd"})
	if err == nil {
		t.Fatal("expected error for absolute path outside CWD, got nil")
	}
	if !isPathOutsideWorkDir(err) {
		t.Fatalf("expected ErrPathOutsideWorkDir, got %v", err)
	}
}

func TestFSHandler_WriteTextFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	content := "written content"
	_, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "out.txt",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("file content = %q, want %q", string(data), content)
	}
}

func TestFSHandler_WriteTextFile_PathTraversal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	_, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "../../tmp/evil.txt",
		Content: "evil",
	})
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !isPathOutsideWorkDir(err) {
		t.Fatalf("expected ErrPathOutsideWorkDir, got %v", err)
	}
}

func TestFSHandler_WriteTextFile_OutsideCWD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	_, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "/tmp/evil.txt",
		Content: "evil",
	})
	if err == nil {
		t.Fatal("expected error for absolute path outside CWD, got nil")
	}
	if !isPathOutsideWorkDir(err) {
		t.Fatalf("expected ErrPathOutsideWorkDir, got %v", err)
	}
}

func TestFSHandler_WriteTextFile_CreatesSubdirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	_, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "sub/dir/file.txt",
		Content: "data",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "sub", "dir", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

// Integration test: roundtrip write then read.
func TestFSHandler_Integration_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	handler := acp.NewFSHandler(dir, nil)

	content := "roundtrip content"
	_, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "round.txt",
		Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := handler.ReadTextFile(context.Background(), acpsdk.ReadTextFileRequest{Path: "round.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != content {
		t.Fatalf("content = %q, want %q", resp.Content, content)
	}
}

func TestFSHandler_WriteTextFile_AppendsRunAuditEntry(t *testing.T) {
	t.Parallel()

	storeRoot := t.TempDir()
	workDir := filepath.Join(storeRoot, "workspace")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	handler := acp.NewFSHandler(workDir, nil, acp.FSAuditMetadata{
		RunID:     "run-123",
		Workflow:  "dev-workflow",
		Step:      "implement",
		Provider:  "claude",
		StoreRoot: storeRoot,
	})

	if _, err := handler.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{
		Path:    "src/main.go",
		Content: "package main",
	}); err != nil {
		t.Fatal(err)
	}

	logData, err := os.ReadFile(filepath.Join(storeRoot, ".orq", "runs", "run-123", "logs", "run.log"))
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	for _, expected := range []string{"acp_filesystem_write", "src/main.go", "\"run_id\":\"run-123\"", "\"step\":\"implement\""} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected %q in audit log, got %q", expected, logText)
		}
	}
}

func isPathOutsideWorkDir(err error) bool {
	return err != nil && containsString(err.Error(), "outside")
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
