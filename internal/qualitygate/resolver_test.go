package qualitygate

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
)

func TestResolveChecks_ResolvesFromToolchain(t *testing.T) {
	selection, err := NewSelection([]CheckKind{CheckTest, CheckLint}, []CheckKind{CheckFmt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toolchain := detect.ToolchainResult{
		"go": detect.ToolchainEntry{Fmt: "gofmt -w .", Test: "go test ./...", Lint: "golangci-lint run"},
	}

	resolved, err := ResolveChecks(selection, toolchain, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("resolved = %v, want 3 checks", resolved)
	}
}

func TestResolveChecks_RequiredCommandMissingFails(t *testing.T) {
	selection, err := NewSelection([]CheckKind{CheckLint}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toolchain := detect.ToolchainResult{
		"python": detect.ToolchainEntry{Test: "pytest"},
	}

	_, err = ResolveChecks(selection, toolchain, "python")
	if !errors.Is(err, ErrCheckUnavailableForStack) {
		t.Fatalf("expected ErrCheckUnavailableForStack, got %v", err)
	}
}

func TestResolveChecks_OptionalCommandMissingIsSkipped(t *testing.T) {
	selection, err := NewSelection([]CheckKind{CheckTest}, []CheckKind{CheckLint})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toolchain := detect.ToolchainResult{
		"python": detect.ToolchainEntry{Test: "pytest"},
	}

	resolved, err := ResolveChecks(selection, toolchain, "python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved = %v, want 1 check (optional lint skipped)", resolved)
	}
}

func TestResolveChecks_UnknownLang(t *testing.T) {
	selection, err := NewSelection([]CheckKind{CheckTest}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = ResolveChecks(selection, detect.ToolchainResult{}, "java")
	if !errors.Is(err, ErrNoStackDetected) {
		t.Fatalf("expected ErrNoStackDetected, got %v", err)
	}
}

func TestPrimaryLang_PrefersDeclaredOrder(t *testing.T) {
	toolchain := detect.ToolchainResult{
		"java": detect.ToolchainEntry{Test: "mvn test"},
		"go":   detect.ToolchainEntry{Test: "go test ./..."},
	}
	lang, ok := PrimaryLang(toolchain)
	if !ok {
		t.Fatalf("expected a primary lang")
	}
	if lang != "go" {
		t.Fatalf("lang = %q, want go (AllLangs order)", lang)
	}
}

func TestPrimaryLang_Empty(t *testing.T) {
	_, ok := PrimaryLang(detect.ToolchainResult{})
	if ok {
		t.Fatalf("expected no primary lang for empty toolchain result")
	}
}
