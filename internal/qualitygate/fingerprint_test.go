package qualitygate

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
)

func TestComputeFingerprint_StableForSameInput(t *testing.T) {
	toolchain := detect.ToolchainResult{"go": detect.ToolchainEntry{Test: "go test ./..."}}
	in := FingerprintInput{TaskID: "10.0", TaskType: "feature", Risk: RiskLow, Lang: "go", Toolchain: toolchain}

	first := ComputeFingerprint(in)
	second := ComputeFingerprint(in)
	if first != second {
		t.Fatalf("fingerprint not stable: %q != %q", first, second)
	}
}

func TestComputeFingerprint_ChangesWhenToolchainChanges(t *testing.T) {
	base := FingerprintInput{
		TaskID:   "10.0",
		TaskType: "feature",
		Risk:     RiskLow,
		Lang:     "go",
		Toolchain: detect.ToolchainResult{
			"go": detect.ToolchainEntry{Test: "go test ./..."},
		},
	}
	changed := base
	changed.Toolchain = detect.ToolchainResult{
		"go": detect.ToolchainEntry{Test: "go test ./... -race"},
	}

	if ComputeFingerprint(base) == ComputeFingerprint(changed) {
		t.Fatalf("fingerprint must change when relevant state changes")
	}
}

func TestComputeFingerprint_ChangesWhenStateDigestChanges(t *testing.T) {
	base := FingerprintInput{TaskID: "10.0", TaskType: "feature", Risk: RiskLow, Lang: "go", StateDigest: "aaa"}
	changed := base
	changed.StateDigest = "bbb"

	if ComputeFingerprint(base) == ComputeFingerprint(changed) {
		t.Fatalf("fingerprint must change when state digest changes")
	}
}
