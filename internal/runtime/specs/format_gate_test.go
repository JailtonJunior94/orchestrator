package specs

import "testing"

func TestNewSpecPanicsOnInvalidFixedArgsFormat(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("newSpec did not panic for FixedArgs with positional item after flag")
		}
	}()

	NewCatalog().newSpec(
		"artificial",
		"Artificial",
		"artificial",
		[]string{"--cwd", "acp"},
		nil,
		"",
		"v0", "0.0.0", "artificial-pkg",
		ContextWindow{},
	)
}

func TestNewSpecPanicsOnInvalidFallbackFixedArgsFormat(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("newSpec did not panic for fallback FixedArgs with positional item after flag")
		}
	}()

	NewCatalog().newSpec(
		"artificial",
		"Artificial",
		"artificial",
		nil,
		[]FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", "artificial-pkg@0.0.0", "--flag", "acp"},
			},
		},
		"",
		"v0", "0.0.0", "artificial-pkg",
		ContextWindow{},
	)
}

func TestValidateFallbackFixedArgsFormatAcceptsNpxPrefixConvention(t *testing.T) {
	t.Parallel()

	valid := [][]string{
		{"--yes", "pkg@1.0.0"},
		{"--yes", "pkg@1.0.0", "acp"},
		{"--yes", "pkg@1.0.0", "--acp"},
	}
	for _, args := range valid {
		if err := ValidateFallbackFixedArgsFormat(args); err != nil {
			t.Errorf("ValidateFallbackFixedArgsFormat(%v) = %v, want nil", args, err)
		}
	}

	invalid := []string{"--yes", "pkg@1.0.0", "--flag", "acp"}
	if err := ValidateFallbackFixedArgsFormat(invalid); err == nil {
		t.Errorf("ValidateFallbackFixedArgsFormat(%v) = nil, want error", invalid)
	}
}
