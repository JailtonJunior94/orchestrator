package durable

type DurabilityPolicy struct{}

var DefaultDurabilityPolicy = DurabilityPolicy{}

func (p DurabilityPolicy) Classify(kind string) Durability {
	switch kind {
	case "declared-section":
		return DurabilityPRD
	default:
		return DurabilityEphemeral
	}
}
