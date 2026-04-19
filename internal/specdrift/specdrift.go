package specdrift

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CoverageResult contains IDs found and missing between source and target.
type CoverageResult struct {
	SourceFile string
	TargetFile string
	FoundIDs   []string
	MissingIDs []string
	Pass       bool
}

// HashResult contains the result of a hash comparison.
type HashResult struct {
	File         string
	ExpectedHash string // hash registered in tasks.md
	ActualHash   string // hash calculated from the spec file
	Match        bool
	NoHashFound  bool // true if tasks.md has no hash comment for this label
}

// DriftReport aggregates coverage and hash results.
type DriftReport struct {
	Coverage []CoverageResult
	Hashes   []HashResult
	Pass     bool
}

var idRegex = regexp.MustCompile(`(?i)(RF-\d+|REQ-\d+)`)

// CheckCoverage extracts RF-nn/REQ-nn IDs from sourceContent and verifies
// their presence in targetContent (case-insensitive).
func CheckCoverage(sourceContent, targetContent []byte) CoverageResult {
	matches := idRegex.FindAllString(string(sourceContent), -1)

	seen := make(map[string]struct{})
	var foundIDs []string
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if _, ok := seen[upper]; !ok {
			seen[upper] = struct{}{}
			foundIDs = append(foundIDs, upper)
		}
	}

	targetStr := strings.ToUpper(string(targetContent))
	var missingIDs []string
	for _, id := range foundIDs {
		if !strings.Contains(targetStr, id) {
			missingIDs = append(missingIDs, id)
		}
	}

	return CoverageResult{
		FoundIDs:   foundIDs,
		MissingIDs: missingIDs,
		Pass:       len(missingIDs) == 0,
	}
}

// CheckHash calculates SHA-256 of specContent and compares it with the hash
// registered in tasksContent via comment <!-- spec-hash-{label}: {hash} -->.
func CheckHash(specContent, tasksContent []byte, label string) HashResult {
	sum := sha256.Sum256(specContent)
	actualHash := fmt.Sprintf("%x", sum)

	pattern := fmt.Sprintf(`<!--\s*spec-hash-%s:\s*([0-9a-f]+)\s*-->`, regexp.QuoteMeta(label))
	re := regexp.MustCompile(pattern)

	match := re.FindSubmatch(tasksContent)
	if match == nil {
		return HashResult{
			File:        label,
			ActualHash:  actualHash,
			NoHashFound: true,
			Match:       false,
		}
	}

	expectedHash := string(match[1])
	return HashResult{
		File:         label,
		ExpectedHash: expectedHash,
		ActualHash:   actualHash,
		Match:        expectedHash == actualHash,
		NoHashFound:  false,
	}
}

// CheckDrift runs both coverage and hash checks for a directory.
// It expects to find prd.md, techspec.md (optional), and tasks.md.
func CheckDrift(dir string) (DriftReport, error) {
	tasksPath := filepath.Join(dir, "tasks.md")
	tasksContent, err := os.ReadFile(tasksPath)
	if err != nil {
		return DriftReport{}, fmt.Errorf("tasks.md not found in %s: %w", dir, err)
	}

	report := DriftReport{Pass: true}

	specs := []struct {
		filename string
		label    string
	}{
		{"prd.md", "prd"},
		{"techspec.md", "techspec"},
	}

	for _, spec := range specs {
		specPath := filepath.Join(dir, spec.filename)
		specContent, err := os.ReadFile(specPath)
		if err != nil {
			// spec file is optional
			continue
		}

		cov := CheckCoverage(specContent, tasksContent)
		cov.SourceFile = spec.filename
		cov.TargetFile = "tasks.md"
		report.Coverage = append(report.Coverage, cov)
		if !cov.Pass {
			report.Pass = false
		}

		hash := CheckHash(specContent, tasksContent, spec.label)
		hash.File = spec.filename
		report.Hashes = append(report.Hashes, hash)
		if !hash.Match {
			report.Pass = false
		}
	}

	return report, nil
}
