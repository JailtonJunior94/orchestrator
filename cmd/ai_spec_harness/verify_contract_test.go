package aispecharness

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestVerifyCommand_ContractPreserved(t *testing.T) {
	cmd := newVerifyCmd()

	if want := "verify [path]"; cmd.Use != want {
		t.Errorf("Use = %q, want %q", cmd.Use, want)
	}
	if cmd.Args == nil {
		t.Error("Args should not be nil: verify accepts at most 1 positional argument")
	}

	wantFlags := []string{"tools", "langs", "source", "global", "by-cli", "check-codex-trust"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing from verify", name)
		}
	}

	var totalFlags int
	cmd.Flags().VisitAll(func(*pflag.Flag) { totalFlags++ })
	if totalFlags != len(wantFlags) {
		t.Errorf("total declared flags = %d, want %d (no new flag added to verify)", totalFlags, len(wantFlags))
	}
}
