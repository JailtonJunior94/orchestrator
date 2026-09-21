package telemetry

var criticalSecurityHooks = map[string]bool{
	"validate-governance":    true,
	"validate-preload":       true,
	"git-operation-gate":     true,
	"validate-task-evidence": true,
}

func IsCriticalSecurityHook(hook string) bool {
	return criticalSecurityHooks[hook]
}

func (c *Catalog) DecideHookAblation(hook string, baselineEntries, withHookEntries []logEntry) AblationResult {
	baseline := c.BuildAblationBaseline(baselineEntries)
	withComponent := c.BuildAblationBaseline(withHookEntries)
	result := c.CompareAblation(hook, ComponentHook, baseline, withComponent)

	if result.Decision == AblationRemove && IsCriticalSecurityHook(hook) {
		result.Decision = AblationKeep
		result.Evidence = append(result.Evidence,
			"critical security hook: REMOVE decision reverted to KEEP regardless of economic return (RF-65)")
	}
	return result
}
