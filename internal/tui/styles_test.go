package tui

import (
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

func TestStepStyleNotEmpty(t *testing.T) {
	t.Parallel()

	m := theme.New()
	statuses := []string{"running", "approved", "waiting_approval", "failed", "skipped", "pending", ""}
	for _, s := range statuses {
		style := StepStyle(m, s)
		// Render a non-empty string to confirm the style does not panic.
		rendered := style.Render("test")
		if rendered == "" {
			t.Errorf("StepStyle(%q).Render returned empty string", s)
		}
	}
}

func TestStepIconCoversAllStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status string
		want   string
	}{
		{"running", iconRunning},
		{"approved", iconApproved},
		{"failed", iconFailed},
		{"skipped", iconSkipped},
		{"waiting_approval", iconWaitingApproval},
		{"pending", iconPending},
		{"", iconPending},
	}
	for _, tc := range cases {
		got := StepIcon(tc.status)
		if got != tc.want {
			t.Errorf("StepIcon(%q) = %q; want %q", tc.status, got, tc.want)
		}
	}
}

func TestPanelStyleRendersNonEmpty(t *testing.T) {
	t.Parallel()

	m := theme.New()
	rendered := Panel(m).Render("content")
	if rendered == "" {
		t.Error("Panel style rendered empty string")
	}
}

func TestTitleStyleRendersNonEmpty(t *testing.T) {
	t.Parallel()

	m := theme.New()
	rendered := Title(m).Render("Maestro")
	if rendered == "" {
		t.Error("Title style rendered empty string")
	}
}

func TestMetadataStyleRendersNonEmpty(t *testing.T) {
	t.Parallel()

	m := theme.New()
	rendered := Metadata(m).Render("v1.0")
	if rendered == "" {
		t.Error("Metadata style rendered empty string")
	}
}
