package capability

type Family string

const (
	FamilyGitPolicy    Family = "git-policy"
	FamilyQualityGate  Family = "quality-gate"
	FamilyEvidenceGate Family = "evidence-gate"
	FamilyCheckpoint   Family = "checkpoint"
	FamilyTelemetry    Family = "telemetry"
)

func Families() []Family {
	return []Family{
		FamilyGitPolicy,
		FamilyQualityGate,
		FamilyEvidenceGate,
		FamilyCheckpoint,
		FamilyTelemetry,
	}
}
