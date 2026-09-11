package precondition

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type Report struct {
	Kind   specs.PreconditionKind
	State  specs.PreconditionState
	Remedy string
}

type CodexTrustChecker func() (specs.PreconditionState, error)

func Evaluate(pre specs.EnforcementPrecondition, projectDir string, copilotReader CopilotConfigReader, codexCheck CodexTrustChecker) Report {
	report := Report{Kind: pre.Kind(), Remedy: pre.Remedy(), State: specs.PreconditionUnknown}

	switch pre.Kind() {
	case specs.PreconditionTrustedFolder:
		state, err := EvaluateCopilotTrustedFolder(copilotReader, projectDir)
		if err != nil {
			report.State = specs.PreconditionUnknown
			return report
		}
		report.State = state
	case specs.PreconditionNoKillSwitch:
		report.State = EvaluateNoKillSwitch(OSEnvironment{}, specs.OpenCodeKillSwitchVars)
	case specs.PreconditionHandshake:
		report.State = EvaluateHandshake()
	case specs.PreconditionTrustedHash:
		if codexCheck == nil {
			report.State = specs.PreconditionUnknown
			return report
		}
		state, err := codexCheck()
		if err != nil {
			report.State = specs.PreconditionUnknown
			return report
		}
		report.State = state
	}
	return report
}
