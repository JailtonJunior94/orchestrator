package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// GeminiAdapter manages install targets for Gemini CLI.
type GeminiAdapter struct {
	*managedAdapter
}

// NewGeminiAdapter creates a Gemini target adapter.
func NewGeminiAdapter(
	fileSystem platform.FileSystem,
	runner platform.CommandRunner,
	projectRoot string,
	globalRoot string,
) *GeminiAdapter {
	verifier := structuralVerifier{
		provider: install.ProviderGemini,
		fs:       fileSystem,
		functionalCheck: func(ctx context.Context, changes []install.PlannedChange, details []string) (install.VerificationStatus, []string, error) {
			expectedSkills := make([]string, 0)
			commandCount := 0
			for _, change := range changes {
				if change.Action() == install.ActionRemove {
					continue
				}
				if change.TargetPath() == "" {
					continue
				}
				switch filepathKind(change.TargetPath()) {
				case install.AssetKindSkill:
					expectedSkills = append(expectedSkills, filepathBaseName(change.TargetPath()))
				case install.AssetKindCommand:
					commandCount++
				}
			}

			statuses := make([]install.VerificationStatus, 0, 2)
			if len(expectedSkills) == 0 {
				statuses = append(statuses, install.VerificationStatusComplete)
			} else {
				result, err := runner.Run(ctx, "gemini", []string{"skills", "list"}, "")
				if err != nil {
					return "", nil, fmt.Errorf("verify provider %q with binary %q: %w", install.ProviderGemini, "gemini", err)
				}

				output := result.Stdout + "\n" + result.Stderr
				missing := make([]string, 0)
				for _, skillName := range expectedSkills {
					if !containsTokenCaseInsensitive(output, skillName) {
						missing = append(missing, skillName)
					}
				}

				if len(missing) > 0 {
					details = append(details, fmt.Sprintf("gemini skills list did not report installed skills: %v", missing))
					statuses = append(statuses, install.VerificationStatusFailed)
				} else {
					details = append(details, "gemini skills list reported all expected skills")
					statuses = append(statuses, install.VerificationStatusComplete)
				}
			}

			if commandCount > 0 {
				details = append(details, "functional verification is limited to Gemini skills; commands remain structurally verified only")
				statuses = append(statuses, install.VerificationStatusPartial)
			}

			return aggregateVerificationStatus(statuses...), details, nil
		},
	}

	return &GeminiAdapter{
		managedAdapter: &managedAdapter{
			provider:        install.ProviderGemini,
			toolDir:         ".gemini",
			projectRoot:     projectRoot,
			globalRoot:      globalRoot,
			fs:              fileSystem,
			runner:          runner,
			verifier:        verifier,
			backupQualifier: "gemini",
		},
	}
}

// ResolveTarget returns the Gemini destination for an asset.
func (a *GeminiAdapter) ResolveTarget(ctx context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error) {
	target, err := a.managedAdapter.ResolveTarget(ctx, scope, asset)
	if err != nil {
		return ResolvedTarget{}, err
	}

	if asset.Kind() == install.AssetKindSkill {
		target.Verification = install.VerificationStatusComplete
	}
	return target, nil
}

func filepathKind(targetPath string) install.AssetKind {
	if containsTokenCaseInsensitive(targetPath, string(filepath.Separator)+"skills"+string(filepath.Separator)) {
		return install.AssetKindSkill
	}
	return install.AssetKindCommand
}

func filepathBaseName(targetPath string) string {
	base := filepath.Base(targetPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
