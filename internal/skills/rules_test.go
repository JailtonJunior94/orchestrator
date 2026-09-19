package skills

import "testing"

func TestUniversalRuleFilesContainsGovernanceAndCodeStyle(t *testing.T) {
	t.Parallel()

	want := map[string]bool{
		"governance.md": false,
		"code-style.md": false,
	}
	for _, ruleFile := range UniversalRuleFiles {
		if _, ok := want[ruleFile]; !ok {
			t.Fatalf("unexpected rule file in UniversalRuleFiles: %q", ruleFile)
		}
		want[ruleFile] = true
	}
	for ruleFile, found := range want {
		if !found {
			t.Errorf("UniversalRuleFiles missing expected entry: %q", ruleFile)
		}
	}
}

func TestUniversalRuleFilesHasNoDuplicates(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, len(UniversalRuleFiles))
	for _, ruleFile := range UniversalRuleFiles {
		if seen[ruleFile] {
			t.Fatalf("duplicate entry in UniversalRuleFiles: %q", ruleFile)
		}
		seen[ruleFile] = true
	}
}
