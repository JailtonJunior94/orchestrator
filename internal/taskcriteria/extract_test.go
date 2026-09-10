package taskcriteria_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/taskcriteria"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantAll     []string
		wantPending []string
	}{
		{
			name:        "definition of done all checked",
			content:     "## Definition of Done\n\n- [x] Item A\n- [x] Item B\n",
			wantAll:     []string{"Item A", "Item B"},
			wantPending: nil,
		},
		{
			name:        "definition of done none checked",
			content:     "## Definition of Done\n\n- [ ] Item A\n- [ ] Item B\n",
			wantAll:     []string{"Item A", "Item B"},
			wantPending: []string{"Item A", "Item B"},
		},
		{
			name:        "mixed checked and unchecked",
			content:     "## Definition of Done\n\n- [x] Done item\n- [ ] Open item\n",
			wantAll:     []string{"Done item", "Open item"},
			wantPending: []string{"Open item"},
		},
		{
			name:        "no relevant section",
			content:     "## Other Section\n\n- [ ] Item A\n",
			wantAll:     nil,
			wantPending: nil,
		},
		{
			name:        "section ends at next heading",
			content:     "## Definition of Done\n\n- [x] Item A\n\n## Next Section\n\n- [ ] Ignored\n",
			wantAll:     []string{"Item A"},
			wantPending: nil,
		},
		{
			name:        "portuguese criterios de sucesso",
			content:     "## Critérios de Sucesso\n\n- [x] Retorna verde\n- [ ] Cobertura alta\n",
			wantAll:     []string{"Retorna verde", "Cobertura alta"},
			wantPending: []string{"Cobertura alta"},
		},
		{
			name:        "acceptance criteria heading",
			content:     "## Acceptance Criteria\n\n- [X] Builds\n- [ ] Lints\n",
			wantAll:     []string{"Builds", "Lints"},
			wantPending: []string{"Lints"},
		},
		{
			name:        "multiple equivalent sections accumulate",
			content:     "## Definition of Done\n\n- [x] First\n\n## Acceptance Criteria\n\n- [ ] Second\n",
			wantAll:     []string{"First", "Second"},
			wantPending: []string{"Second"},
		},
		{
			name:        "blank checklist text is skipped",
			content:     "## Definition of Done\n\n- [ ] \n- [x] Real item\n",
			wantAll:     []string{"Real item"},
			wantPending: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotAll := taskcriteria.Extract([]byte(tc.content))
			if !equal(gotAll, tc.wantAll) {
				t.Errorf("Extract() = %v, want %v", gotAll, tc.wantAll)
			}
			gotPending := taskcriteria.Pending([]byte(tc.content))
			if !equal(gotPending, tc.wantPending) {
				t.Errorf("Pending() = %v, want %v", gotPending, tc.wantPending)
			}
		})
	}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestExtractIgnoresChecklistOutsideSection(t *testing.T) {
	content := strings.Join([]string{
		"# Task",
		"",
		"## Overview",
		"- [ ] not a criterion",
		"",
		"## Definition of Done",
		"- [ ] the only criterion",
		"",
		"## Files",
		"- [ ] also not a criterion",
	}, "\n")

	all := taskcriteria.Extract([]byte(content))
	if len(all) != 1 || all[0] != "the only criterion" {
		t.Fatalf("Extract() = %v, want [the only criterion]", all)
	}
}
