package aispecharness

import (
	"testing"
)

func TestUpgradeRef_MutuallyExclusiveWithSource(t *testing.T) {
	upgradeRef = "v1.1.0"
	upgradeSource = "/tmp/some-source"
	t.Cleanup(func() {
		upgradeRef = ""
		upgradeSource = ""
	})

	err := runUpgrade(upgradeCmd, []string{"/tmp/project"})
	if err == nil {
		t.Fatal("expected error when --ref and --source are both set")
	}
	if err.Error() != "--ref e --source sao mutuamente exclusivos" {
		t.Fatalf("unexpected error message: %v", err)
	}
}
