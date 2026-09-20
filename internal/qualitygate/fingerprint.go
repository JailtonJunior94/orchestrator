package qualitygate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
)

type FingerprintInput struct {
	TaskID      string
	TaskType    TaskType
	Risk        Risk
	Lang        string
	Toolchain   detect.ToolchainResult
	StateDigest string
}

type fingerprintPayload struct {
	TaskID      string                           `json:"task_id"`
	TaskType    string                           `json:"task_type"`
	Risk        string                           `json:"risk"`
	Lang        string                           `json:"lang"`
	Toolchain   map[string]detect.ToolchainEntry `json:"toolchain"`
	StateDigest string                           `json:"state_digest"`
}

func ComputeFingerprint(in FingerprintInput) string {
	payload := fingerprintPayload{
		TaskID:      in.TaskID,
		TaskType:    string(in.TaskType),
		Risk:        in.Risk.String(),
		Lang:        in.Lang,
		Toolchain:   in.Toolchain,
		StateDigest: in.StateDigest,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Errorf("qualitygate: encode fingerprint payload: %w", err))
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
