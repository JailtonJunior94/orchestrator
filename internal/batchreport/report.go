package batchreport

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/tracking"
	"github.com/JailtonJunior94/ai-spec-harness/internal/txn"
)

type FileOutcome string

const (
	OutcomeCreated   FileOutcome = "created"
	OutcomeUpdated   FileOutcome = "updated"
	OutcomePreserved FileOutcome = "preserved"
	OutcomeMerged    FileOutcome = "merged"
	OutcomeConflict  FileOutcome = "conflict"
)

type Report struct {
	Outcomes map[string]FileOutcome
}

func Build(tracker *tracking.Tracker) *Report {
	outcomes := make(map[string]FileOutcome, len(tracker.Outcomes()))

	for path, state := range tracker.Outcomes() {
		switch state {
		case txn.StateCreated:
			outcomes[path] = OutcomeCreated
		case txn.StateUpdated:
			outcomes[path] = OutcomeUpdated
		case txn.StatePreserved:
			outcomes[path] = OutcomePreserved
		}
	}

	for _, path := range tracker.MergedPaths() {
		outcomes[path] = OutcomeMerged
	}

	for _, conflict := range tracker.Conflicts() {
		outcomes[conflict.Path] = OutcomeConflict
	}

	return &Report{Outcomes: outcomes}
}

func (r *Report) CountsByOutcome() map[FileOutcome]int {
	counts := make(map[FileOutcome]int, 5)
	for _, outcome := range r.Outcomes {
		counts[outcome]++
	}
	return counts
}

func (r *Report) PathsByOutcome(outcome FileOutcome) []string {
	var out []string
	for path, o := range r.Outcomes {
		if o == outcome {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func Print(printer *output.Printer, label string, report *Report) {
	counts := report.CountsByOutcome()
	printer.Info("")
	printer.Info("Batch summary (%s): created=%d updated=%d preserved=%d merged=%d conflict=%d",
		label, counts[OutcomeCreated], counts[OutcomeUpdated], counts[OutcomePreserved], counts[OutcomeMerged], counts[OutcomeConflict])

	for _, path := range report.PathsByOutcome(OutcomeConflict) {
		printer.Warn("  overwritten (conflict, --overwrite-conflicts): %s", path)
	}
}

func ConflictError(conflicts []txn.Conflict, sourceDir string) error {
	lines := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		lines = append(lines, fmt.Sprintf("  - %s (checksum diverges from manifest; canonical origin: %s)", c.Path, filepath.Join(sourceDir, c.Path)))
	}
	return fmt.Errorf(
		"batch aborted: %d managed file(s) in conflict (manually edited after the previous installation). "+
			"Nothing was changed. Move the customization to the canonical origin at %s or "+
			"use --overwrite-conflicts to overwrite by naming each file:\n%s",
		len(conflicts), sourceDir, strings.Join(lines, "\n"),
	)
}
