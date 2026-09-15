package reviewverdict

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const FixturePath = "../../tests/fixtures/approval/verdict-tokens.tsv"

type FixtureCase struct {
	Line string
	Want string
}

func LoadFixture(t *testing.T, path string) []FixtureCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)

	var cases []FixtureCase
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		text, want, found := strings.Cut(line, "\t")
		require.True(t, found, "fixture line without tab separator: %q", line)
		cases = append(cases, FixtureCase{Line: text, Want: strings.TrimSpace(want)})
	}
	require.NotEmpty(t, cases)
	return cases
}

func TestParseMatchesTheSharedVerdictFixture(t *testing.T) {
	for _, tc := range LoadFixture(t, FixturePath) {
		t.Run(tc.Line, func(t *testing.T) {
			verdict, _ := Parse(tc.Line)
			want := tc.Want
			if want == "NONE" {
				want = ""
			}
			require.Equal(t, want, verdict)
		})
	}
}
