package taskloop

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ErrEvidenceMissing indica que a reescrita de evidencia falhou.
var ErrEvidenceMissing = errors.New("taskloop: evidencia ausente")

const (
	evidenceStart = "<!-- evidence:start -->"
	evidenceEnd   = "<!-- evidence:end -->"
)

// EvidenceRecorder persiste AcceptanceReport no arquivo da task de forma idempotente.
type EvidenceRecorder interface {
	Append(ctx context.Context, taskFile string, report AcceptanceReport) error
}

type defaultEvidenceRecorder struct {
	fsys fs.FileSystem
}

// NewEvidenceRecorder cria um EvidenceRecorder com o filesystem fornecido.
func NewEvidenceRecorder(fsys fs.FileSystem) EvidenceRecorder {
	return &defaultEvidenceRecorder{fsys: fsys}
}

// Append escreve ou substitui o bloco de evidência no arquivo da task.
// Operação idempotente: aplicar duas vezes produz arquivo equivalente.
func (r *defaultEvidenceRecorder) Append(_ context.Context, taskFile string, report AcceptanceReport) error {
	data, err := r.fsys.ReadFile(taskFile)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEvidenceMissing, err)
	}

	block := buildEvidenceBlock(report)
	updated := replaceEvidenceBlock(data, block)

	if err := r.fsys.WriteFile(taskFile, updated); err != nil {
		return fmt.Errorf("%w: %w", ErrEvidenceMissing, err)
	}
	return nil
}

func buildEvidenceBlock(r AcceptanceReport) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\n", evidenceStart)
	fmt.Fprintf(&b, "## Evidência\n\n")
	fmt.Fprintf(&b, "**Task:** %s\n", r.TaskID)
	fmt.Fprintf(&b, "**Passed:** %v\n", r.Passed)
	fmt.Fprintf(&b, "**Critérios:** %d/%d\n\n", r.CriteriaMet, r.CriteriaTotal)
	if r.GoTestOutput != "" {
		fmt.Fprintf(&b, "### go test\n\n```\n%s\n```\n\n", r.GoTestOutput)
	}
	if r.GoVetOutput != "" {
		fmt.Fprintf(&b, "### go vet\n\n```\n%s\n```\n\n", r.GoVetOutput)
	}
	if r.LintOutput != "" {
		fmt.Fprintf(&b, "### lint\n\n```\n%s\n```\n\n", r.LintOutput)
	}
	fmt.Fprintf(&b, "%s\n", evidenceEnd)
	return b.Bytes()
}

func replaceEvidenceBlock(data, block []byte) []byte {
	startMarker := []byte(evidenceStart)
	endMarker := []byte(evidenceEnd)

	startIdx := bytes.Index(data, startMarker)
	endIdx := bytes.Index(data, endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		// Bloco ausente: anexar ao final preservando newline
		trimmed := bytes.TrimRight(data, "\n")
		var b bytes.Buffer
		b.Write(trimmed)
		b.WriteByte('\n')
		b.Write(block)
		return b.Bytes()
	}

	// Substituir bloco existente (incluindo marcadores)
	endOfBlock := endIdx + len(endMarker)
	var b bytes.Buffer
	b.Write(data[:startIdx])
	b.Write(block)
	// Preservar conteúdo após o bloco
	after := data[endOfBlock:]
	if len(after) > 0 && after[0] == '\n' {
		after = after[1:]
	}
	if len(after) > 0 {
		b.WriteByte('\n')
		b.Write(after)
	}
	return b.Bytes()
}
