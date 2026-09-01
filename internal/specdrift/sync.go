package specdrift

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

// SyncSpecHash recomputa os SHA-256 de prd.md e techspec.md encontrados no
// mesmo diretorio de tasksPath. Comentarios spec-hash antigos, duplicados ou
// placeholders sao removidos antes de inserir exatamente um comentario correto
// por spec existente.
func (c *Catalog) SyncSpecHash(tasksPath string) error {
	dir := filepath.Dir(tasksPath)
	if err := c.guardApprovedState(dir); err != nil {
		return err
	}

	tasksBytes, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("ler tasks.md: %w", err)
	}

	updated := string(tasksBytes)
	var toInsert []string

	specs := []struct{ filename, label string }{
		{"prd.md", "prd"},
		{"techspec.md", "techspec"},
	}

	for _, spec := range specs {
		pattern := fmt.Sprintf(`(?m)^\s*<!--\s*spec-hash-%s:\s*[^>]*-->\s*\r?\n?`, regexp.QuoteMeta(spec.label))
		re := regexp.MustCompile(pattern)
		updated = re.ReplaceAllString(updated, "")

		specBytes, err := os.ReadFile(filepath.Join(dir, spec.filename))
		if err != nil {
			continue
		}

		sum := sha256.Sum256(specBytes)
		hash := fmt.Sprintf("%x", sum)
		toInsert = append(toInsert, fmt.Sprintf("<!-- spec-hash-%s: %s -->", spec.label, hash))
	}

	if len(toInsert) > 0 {
		updated = strings.Join(toInsert, "\n") + "\n" + updated
	}

	if updated == string(tasksBytes) {
		return nil
	}

	return os.WriteFile(tasksPath, []byte(updated), 0o644)
}

func (c *Catalog) guardApprovedState(dir string) error {
	store := sdd.NewStore()
	state, err := store.Load(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("validar estado SDD antes de sincronizar: %w", err)
	}
	for _, artifact := range []struct {
		kind sdd.Artifact
		file string
	}{
		{kind: sdd.ArtifactPRD, file: "prd.md"},
		{kind: sdd.ArtifactTechSpec, file: "techspec.md"},
	} {
		entry := state.Artifacts[artifact.kind]
		if !entry.Approved {
			continue
		}
		digest, digestErr := store.DigestFile(filepath.Join(dir, artifact.file))
		if digestErr != nil {
			return digestErr
		}
		if digest != entry.SHA256 {
			if _, invalidateErr := store.Invalidate(dir, artifact.kind); invalidateErr != nil {
				return invalidateErr
			}
			return fmt.Errorf("%s aprovado foi alterado; downstream marcado stale, reprove antes de sincronizar", artifact.kind)
		}
	}
	return nil
}
