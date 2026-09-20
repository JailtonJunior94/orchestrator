package hookpolicy_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookpolicy"
	"github.com/stretchr/testify/require"
)

func baseContract() harness.Contract {
	return harness.Contract{
		Version: harness.SupportedVersion,
		Git: harness.GitPolicy{
			AutoCommit: false,
			AutoPush:   false,
		},
		Approval: harness.ApprovalPolicy{RequireForDestructiveOperations: true},
		Quality:  harness.QualityPolicy{RequireTests: true, RequireLint: true},
		Evidence: harness.EvidencePolicy{RequireExecutionReport: true},
		Skills:   harness.SkillsPolicy{DiscoveryMode: "declared"},
	}
}

func subcommands(t *testing.T, scope hookpolicy.GitScope) []string {
	t.Helper()
	operations := scope.Operations()
	found := make([]string, 0, len(operations))
	for _, operation := range operations {
		found = append(found, operation.Subcommand())
	}
	return found
}

func TestGitScope_DerivesFromContract(t *testing.T) {
	scenarios := []struct {
		name          string
		mutate        func(contract *harness.Contract)
		wantCommit    bool
		wantPush      bool
		wantForcePush bool
	}{
		{
			name:          "auto_commit false keeps commit gated",
			mutate:        func(contract *harness.Contract) { contract.Git.AutoCommit = false },
			wantCommit:    true,
			wantPush:      true,
			wantForcePush: true,
		},
		{
			name:          "auto_commit true removes commit from scope",
			mutate:        func(contract *harness.Contract) { contract.Git.AutoCommit = true },
			wantCommit:    false,
			wantPush:      true,
			wantForcePush: true,
		},
		{
			name:          "auto_push true removes push and push force from scope",
			mutate:        func(contract *harness.Contract) { contract.Git.AutoPush = true },
			wantCommit:    true,
			wantPush:      false,
			wantForcePush: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			contract := baseContract()
			scenario.mutate(&contract)

			scope, err := hookpolicy.NewGitScope(contract)
			require.NoError(t, err)

			found := subcommands(t, scope)
			require.Equal(t, scenario.wantCommit, contains(found, "commit"))
			require.Equal(t, scenario.wantPush, contains(found, "push"))
			require.Equal(t, scenario.wantForcePush, contains(found, "push --force"))
		})
	}
}

func TestGitScope_AlwaysIncludesDestructiveOperations(t *testing.T) {
	contract := baseContract()
	contract.Git.AutoCommit = true
	contract.Git.AutoPush = true

	scope, err := hookpolicy.NewGitScope(contract)
	require.NoError(t, err)

	found := subcommands(t, scope)
	for _, want := range []string{"reset --hard", "clean", "checkout --", "restore"} {
		require.True(t, contains(found, want), "expected %q in destructive scope, got %v", want, found)
	}
}

func TestGitScope_DestructiveOperationsAreMarkedDestructiveAndApprovable(t *testing.T) {
	scope, err := hookpolicy.NewGitScope(baseContract())
	require.NoError(t, err)

	for _, operation := range scope.Operations() {
		switch operation.Subcommand() {
		case "reset --hard", "clean", "checkout --", "restore", "push --force":
			require.True(t, operation.Destructive(), "%s should be destructive", operation.Subcommand())
		case "commit", "push":
			require.False(t, operation.Destructive(), "%s should not be destructive", operation.Subcommand())
		}
		require.True(t, operation.RequiresApproval(), "%s should require approval", operation.Subcommand())
	}
}

func TestGitScope_FingerprintChangesWithPolicy(t *testing.T) {
	baseScope, err := hookpolicy.NewGitScope(baseContract())
	require.NoError(t, err)
	baseFingerprint := baseScope.Fingerprint()
	require.NotEmpty(t, baseFingerprint)

	changed := baseContract()
	changed.Git.AutoCommit = true
	changedScope, err := hookpolicy.NewGitScope(changed)
	require.NoError(t, err)

	require.NotEqual(t, baseFingerprint, changedScope.Fingerprint())
}

func TestGitScope_FingerprintIsDeterministic(t *testing.T) {
	contract := baseContract()

	first, err := hookpolicy.NewGitScope(contract)
	require.NoError(t, err)
	second, err := hookpolicy.NewGitScope(contract)
	require.NoError(t, err)

	require.Equal(t, first.Fingerprint(), second.Fingerprint())
}

func TestGitScope_FingerprintIsStableAcrossOperationOrder(t *testing.T) {
	contract := baseContract()

	scope, err := hookpolicy.NewGitScope(contract)
	require.NoError(t, err)

	operations := scope.Operations()
	require.True(t, len(operations) > 1)
	for i := 1; i < len(operations); i++ {
		require.LessOrEqual(t, operations[i-1].Subcommand(), operations[i].Subcommand())
	}
}

func TestNewGitOperation_RejectsEmptySubcommand(t *testing.T) {
	_, err := hookpolicy.NewGitOperation("", false, true)
	require.Error(t, err)
}

func TestNewGitScope_RejectsUnsupportedVersion(t *testing.T) {
	contract := baseContract()
	contract.Version = harness.SupportedVersion + 1

	_, err := hookpolicy.NewGitScope(contract)
	require.Error(t, err)
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
