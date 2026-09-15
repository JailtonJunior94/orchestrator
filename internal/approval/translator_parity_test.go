package approval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const verdictFixturePath = "../../tests/fixtures/approval/verdict-tokens.tsv"

type verdictFixtureCase struct {
	line string
	want string
}

func loadVerdictFixture(t *testing.T) []verdictFixtureCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(verdictFixturePath))
	require.NoError(t, err)

	var cases []verdictFixtureCase
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		text, want, found := strings.Cut(line, "\t")
		require.True(t, found, "fixture line without tab separator: %q", line)
		cases = append(cases, verdictFixtureCase{line: text, want: strings.TrimSpace(want)})
	}
	require.NotEmpty(t, cases)
	return cases
}

func TestTranslatorMatchesTheSharedVerdictFixture(t *testing.T) {
	for _, tc := range loadVerdictFixture(t) {
		t.Run(tc.line, func(t *testing.T) {
			want := tc.want
			if want == "NONE" {
				want = VerdictBlocked.String()
			}
			require.Equal(t, want, NewTranslator().Translate(tc.line+"\n").String())
		})
	}
}
