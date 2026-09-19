package harness

const SupportedVersion = 1

type Source string

const (
	SourceFile    Source = "file"
	SourceDefault Source = "default"
)

type GitPolicy struct {
	AutoCommit bool `json:"auto_commit"`
	AutoPush   bool `json:"auto_push"`
}

type ApprovalPolicy struct {
	RequireForDestructiveOperations bool `json:"require_for_destructive_operations"`
}

type QualityPolicy struct {
	RequireTests bool `json:"require_tests"`
	RequireLint  bool `json:"require_lint"`
}

type EvidencePolicy struct {
	RequireExecutionReport bool `json:"require_execution_report"`
}

type SkillsPolicy struct {
	DiscoveryMode string `json:"discovery_mode"`
}

type Contract struct {
	Version  int            `json:"version"`
	Git      GitPolicy      `json:"git"`
	Approval ApprovalPolicy `json:"approval"`
	Quality  QualityPolicy  `json:"quality"`
	Evidence EvidencePolicy `json:"evidence"`
	Skills   SkillsPolicy   `json:"skills"`
}
