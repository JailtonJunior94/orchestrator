package specdrift

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SyncSpecHash recomputes SHA-256 hashes for prd.md and techspec.md found in
// the same directory as tasksPath and updates or inserts the
// <!-- spec-hash-{label}: {hash} --> comments in tasks.md.
//
// Files that do not exist are silently skipped (they are optional).
// Returns an error only when tasks.md cannot be read or written.
func SyncSpecHash(tasksPath string) error {
	dir := filepath.Dir(tasksPath)

	tasksBytes, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("reading tasks.md: %w", err)
	}

	updated := string(tasksBytes)
	var toInsert []string

	specs := []struct{ filename, label string }{
		{"prd.md", "prd"},
		{"techspec.md", "techspec"},
	}

	for _, spec := range specs {
		specBytes, err := os.ReadFile(filepath.Join(dir, spec.filename))
		if err != nil {
			continue // optional — skip when absent
		}

		sum := sha256.Sum256(specBytes)
		hash := fmt.Sprintf("%x", sum)
		comment := fmt.Sprintf("<!-- spec-hash-%s: %s -->", spec.label, hash)

		pattern := fmt.Sprintf(`<!--\s*spec-hash-%s:\s*[0-9a-f]+\s*-->`, regexp.QuoteMeta(spec.label))
		re := regexp.MustCompile(pattern)

		if re.MatchString(updated) {
			updated = re.ReplaceAllString(updated, comment)
		} else {
			toInsert = append(toInsert, comment)
		}
	}

	if len(toInsert) > 0 {
		updated = strings.Join(toInsert, "\n") + "\n" + updated
	}

	if updated == string(tasksBytes) {
		return nil // nothing changed
	}

	return os.WriteFile(tasksPath, []byte(updated), 0o644)
}
