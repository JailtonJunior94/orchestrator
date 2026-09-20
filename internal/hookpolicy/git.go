package hookpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
)

type GitOperation struct {
	subcommand       string
	destructive      bool
	requiresApproval bool
}

func NewGitOperation(subcommand string, destructive, requiresApproval bool) (GitOperation, error) {
	if subcommand == "" {
		return GitOperation{}, fmt.Errorf("hookpolicy: git operation requires a non-empty subcommand")
	}
	return GitOperation{
		subcommand:       subcommand,
		destructive:      destructive,
		requiresApproval: requiresApproval,
	}, nil
}

func (o GitOperation) Subcommand() string {
	return o.subcommand
}

func (o GitOperation) Destructive() bool {
	return o.destructive
}

func (o GitOperation) RequiresApproval() bool {
	return o.requiresApproval
}

type GitScope interface {
	Operations() []GitOperation
	Fingerprint() string
}

type gitScope struct {
	operations []GitOperation
}

var _ GitScope = (*gitScope)(nil)

func NewGitScope(contract harness.Contract) (GitScope, error) {
	if contract.Version != harness.SupportedVersion {
		return nil, fmt.Errorf("hookpolicy: unsupported harness contract version %d", contract.Version)
	}

	operations := make([]GitOperation, 0, 6)

	if !contract.Git.AutoCommit {
		operation, err := NewGitOperation("commit", false, true)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}

	if !contract.Git.AutoPush {
		push, err := NewGitOperation("push", false, true)
		if err != nil {
			return nil, err
		}
		forcePush, err := NewGitOperation("push --force", true, true)
		if err != nil {
			return nil, err
		}
		operations = append(operations, push, forcePush)
	}

	destructive := []string{"reset --hard", "clean", "checkout --", "restore"}
	for _, subcommand := range destructive {
		operation, err := NewGitOperation(subcommand, true, true)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}

	sort.Slice(operations, func(i, j int) bool {
		return operations[i].subcommand < operations[j].subcommand
	})

	return &gitScope{operations: operations}, nil
}

func (s *gitScope) Operations() []GitOperation {
	cloned := make([]GitOperation, len(s.operations))
	copy(cloned, s.operations)
	return cloned
}

func (s *gitScope) Fingerprint() string {
	type fingerprintOperation struct {
		Subcommand       string `json:"subcommand"`
		Destructive      bool   `json:"destructive"`
		RequiresApproval bool   `json:"requires_approval"`
	}

	payload := make([]fingerprintOperation, 0, len(s.operations))
	for _, operation := range s.operations {
		payload = append(payload, fingerprintOperation{
			Subcommand:       operation.subcommand,
			Destructive:      operation.destructive,
			RequiresApproval: operation.requiresApproval,
		})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Errorf("hookpolicy: encode git scope for fingerprint: %w", err))
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
