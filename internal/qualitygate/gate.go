package qualitygate

import (
	"context"
	"fmt"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

const GateID = "quality-gate"

type EvaluationInput struct {
	ProjectDir  string
	TaskID      string
	TaskType    TaskType
	Risk        Risk
	StateDigest string
}

type ToolchainProvider interface {
	Detect(projectDir string) detect.ToolchainResult
}

type Gate struct {
	policies  PolicyLoader
	toolchain ToolchainProvider
	executor  Executor
	cache     Cache
	evidence  EvidenceWriter
}

func NewGate(policies PolicyLoader, toolchain ToolchainProvider, executor Executor, cache Cache, evidence EvidenceWriter) *Gate {
	return &Gate{
		policies:  policies,
		toolchain: toolchain,
		executor:  executor,
		cache:     cache,
		evidence:  evidence,
	}
}

func (g *Gate) Evaluate(ctx context.Context, in EvaluationInput) hookcontract.Result {
	policySet, err := g.policies.Load(in.ProjectDir)
	if err != nil {
		return errorResult(fmt.Sprintf("load policy: %s", err))
	}

	selection, err := policySet.Lookup(in.TaskType, in.Risk)
	if err != nil {
		return errorResult(err.Error())
	}

	policyID := gatePolicyID(in.TaskType, in.Risk)

	toolchainResult := g.toolchain.Detect(in.ProjectDir)
	lang, ok := PrimaryLang(toolchainResult)
	if !ok {
		return errorResultWithPolicy(policyID, "qualitygate: no stack detected for project")
	}

	fingerprint := ComputeFingerprint(FingerprintInput{
		TaskID:      in.TaskID,
		TaskType:    in.TaskType,
		Risk:        in.Risk,
		Lang:        lang,
		Toolchain:   toolchainResult,
		StateDigest: in.StateDigest,
	})

	if g.cache != nil {
		if entry, hit, cacheErr := g.cache.Get(in.TaskID); cacheErr == nil && hit && entry.Fingerprint == fingerprint && isSkippableDecision(entry.Decision) {
			if skip, ok := notApplicableSkip(policyID, entry); ok {
				return skip
			}
		}
	}

	resolved, err := ResolveChecks(selection, toolchainResult, lang)
	if err != nil {
		return errorResultWithPolicy(policyID, err.Error())
	}

	outcome := g.runChecks(ctx, in.ProjectDir, policyID, resolved)
	verdict := NewVerdict(outcome.result)
	final := verdict.Result()

	infraWarnings := make(map[string]string, 2)

	if g.evidence != nil {
		if persistErr := g.evidence.Persist(buildEvidence(in, policyID, fingerprint, final, outcome.checks)); persistErr != nil {
			infraWarnings["evidence_persist_error"] = persistErr.Error()
		}
	}

	if g.cache != nil {
		if putErr := g.cache.Put(in.TaskID, CacheEntry{
			Fingerprint: fingerprint,
			Decision:    final.Decision().String(),
			Reason:      final.Reason(),
			PolicyID:    final.PolicyID(),
			GateID:      final.GateID(),
		}); putErr != nil {
			infraWarnings["cache_put_error"] = putErr.Error()
		}
	}

	if len(infraWarnings) > 0 {
		final = final.WithMetadata(infraWarnings)
	}

	return final
}

type checkOutcome struct {
	result hookcontract.Result
	checks []CheckEvidence
}

func (g *Gate) runChecks(ctx context.Context, dir, policyID string, checks []ResolvedCheck) checkOutcome {
	var failedRequired []string
	var failedOptional []string
	evidence := make([]CheckEvidence, 0, len(checks))

	for _, check := range checks {
		execResult, err := g.executor.Execute(ctx, dir, check.Command)
		failed := err != nil || execResult.ExitCode != 0
		evidence = append(evidence, CheckEvidence{
			Kind:     check.Kind.String(),
			Command:  check.Command,
			Required: check.Required,
			ExitCode: execResult.ExitCode,
			Output:   execResult.Output,
		})
		if failed {
			label := fmt.Sprintf("%s (%s)", check.Kind, check.Command)
			if check.Required {
				failedRequired = append(failedRequired, label)
			} else {
				failedOptional = append(failedOptional, label)
			}
		}
	}

	if len(failedRequired) > 0 {
		reason := fmt.Sprintf("required checks failed: %s", strings.Join(failedRequired, "; "))
		result, err := hookcontract.NewResult(hookcontract.DecisionBlock, reason, policyID, GateID)
		if err != nil {
			return checkOutcome{result: errorResultWithPolicy(policyID, err.Error()), checks: evidence}
		}
		return checkOutcome{result: result, checks: evidence}
	}

	if len(failedOptional) > 0 {
		reason := fmt.Sprintf("optional checks failed: %s", strings.Join(failedOptional, "; "))
		result, err := hookcontract.NewResult(hookcontract.DecisionWarn, reason, policyID, GateID)
		if err != nil {
			return checkOutcome{result: errorResultWithPolicy(policyID, err.Error()), checks: evidence}
		}
		return checkOutcome{result: result, checks: evidence}
	}

	return checkOutcome{result: hookcontract.NewAllow(), checks: evidence}
}

func gatePolicyID(taskType TaskType, risk Risk) string {
	return fmt.Sprintf("quality-gate:%s:%s", taskType, risk)
}

func errorResult(reason string) hookcontract.Result {
	return errorResultWithPolicy("", reason)
}

func errorResultWithPolicy(policyID, reason string) hookcontract.Result {
	result, err := hookcontract.NewResult(hookcontract.DecisionError, reason, policyID, GateID)
	if err != nil {
		return hookcontract.NewNotApplicable(reason)
	}
	return result
}

func isSkippableDecision(decision string) bool {
	switch decision {
	case hookcontract.DecisionAllow.String(), hookcontract.DecisionWarn.String():
		return true
	default:
		return false
	}
}

func notApplicableSkip(policyID string, entry CacheEntry) (hookcontract.Result, bool) {
	reason := "qualitygate: skip - fingerprint unchanged since last run"
	result, err := hookcontract.NewResult(hookcontract.DecisionNotApplicable, reason, policyID, GateID)
	if err != nil {
		return hookcontract.Result{}, false
	}
	result = result.WithMetadata(map[string]string{
		"previous_decision": entry.Decision,
		"previous_reason":   entry.Reason,
	})
	return result, true
}

func buildEvidence(in EvaluationInput, policyID, fingerprint string, result hookcontract.Result, checks []CheckEvidence) GateEvidence {
	return GateEvidence{
		GateID:      GateID,
		PolicyID:    policyID,
		TaskID:      in.TaskID,
		TaskType:    string(in.TaskType),
		Risk:        in.Risk.String(),
		Fingerprint: fingerprint,
		Decision:    result.Decision().String(),
		Reason:      result.Reason(),
		Checks:      checks,
	}
}
