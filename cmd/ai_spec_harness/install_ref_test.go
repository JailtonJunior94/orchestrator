package aispecharness

import (
	"testing"
)

func TestInstallRef_MutuallyExclusiveWithSource(t *testing.T) {
	installRef = "v1.0.0"
	installSource = "/tmp/some-source"
	t.Cleanup(func() {
		installRef = ""
		installSource = ""
	})

	err := runInstall(installCmd, []string{"/tmp/project"})
	if err == nil {
		t.Fatal("expected error when --ref and --source are both set")
	}
	if err.Error() != "--ref e --source sao mutuamente exclusivos" {
		t.Fatalf("unexpected error message: %v", err)
	}
}
