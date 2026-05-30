package aispecharness

import "testing"

func TestUpgradeRef_MutuallyExclusiveWithSource(t *testing.T) {
	cmd := newUpgradeCmd()
	if err := cmd.Flags().Set("ref", "v1.1.0"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("source", "/tmp/some-source"); err != nil {
		t.Fatal(err)
	}

	err := (&upgradeCommand{}).run(cmd, []string{"/tmp/project"})
	if err == nil {
		t.Fatal("expected error when --ref and --source are both set")
	}
	if err.Error() != "--ref e --source sao mutuamente exclusivos" {
		t.Fatalf("unexpected error message: %v", err)
	}
}
