package capability

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/parity"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type State string

const (
	StateSupported          State = "supported"
	StateUnsupported        State = "unsupported"
	StateProviderCapability State = "provider capability"
	StateUnknown            State = "unknown"
)

const EvidenceTest = "TestParitySuite/TestParity_AllTools"

type Cell struct {
	Provider    string `json:"provider"`
	Capability  string `json:"capability"`
	Description string `json:"description"`
	State       State  `json:"state"`
	Reason      string `json:"reason,omitempty"`
	Test        string `json:"test,omitempty"`
}

type Matrix struct {
	Cells []Cell `json:"cells"`
}

const capabilityProjectDir = "/capability-matrix-source"

func Generate() (Matrix, error) {
	checker := parity.NewChecker()
	providers := parity.CanonicalProviders()

	snap, err := checker.Generate(capabilityProjectDir, providers, nil, "full")
	if err != nil {
		return Matrix{}, fmt.Errorf("generate parity snapshot: %w", err)
	}

	invariants := checker.Invariants()
	results := checker.Run(snap, invariants)

	var cells []Cell
	for _, cr := range results {
		if cr.Skipped {
			continue
		}
		for _, provider := range resolveProviders(cr.Invariant) {
			cells = append(cells, buildCell(provider, cr))
		}
	}

	if len(cells) == 0 {
		return Matrix{}, errors.New("generated capability matrix has zero cells")
	}

	sortCells(cells)
	return Matrix{Cells: cells}, nil
}

func resolveProviders(inv *parity.Invariant) []skills.Tool {
	if len(inv.AppliesTo) == 0 {
		return parity.CanonicalProviders()
	}
	return inv.AppliesTo
}

func buildCell(provider skills.Tool, cr parity.CheckResult) Cell {
	inv := cr.Invariant
	cell := Cell{
		Provider:    string(provider),
		Capability:  inv.ID,
		Description: inv.Description,
	}

	if inv.EvidenceInvalid() {
		cell.State = StateUnknown
		cell.Reason = fmt.Sprintf("invariant satisfied by generator-injected stub %q; auto-satisfied by construction and never sustains a supported cell (V-25)", strings.Join(inv.SelfSatisfiedStubPaths, ", "))
		return cell
	}

	if !cr.Result.OK {
		cell.State = StateUnsupported
		cell.Reason = cr.Result.Reason
		return cell
	}

	if inv.Scope() == parity.ScopeUniversal {
		cell.State = StateSupported
	} else {
		cell.State = StateProviderCapability
	}
	cell.Test = EvidenceTest
	return cell
}

func sortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Capability != cells[j].Capability {
			return cells[i].Capability < cells[j].Capability
		}
		return providerOrder(cells[i].Provider) < providerOrder(cells[j].Provider)
	})
}

func providerOrder(provider string) int {
	for i, t := range parity.CanonicalProviders() {
		if string(t) == provider {
			return i
		}
	}
	return len(parity.CanonicalProviders())
}

func RenderBoth(m Matrix) (jsonBytes []byte, markdownBytes []byte, err error) {
	jsonBytes, err = RenderJSON(m)
	if err != nil {
		return nil, nil, err
	}
	return jsonBytes, RenderMarkdown(m), nil
}

func RenderJSON(m Matrix) ([]byte, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal capability matrix: %w", err)
	}
	return append(data, '\n'), nil
}

func RenderMarkdown(m Matrix) []byte {
	var b strings.Builder
	b.WriteString("# Capability Matrix\n\n")
	b.WriteString("Gerado a partir dos invariantes de `internal/parity` (RF-18). Nao editar a mao: rode\n")
	b.WriteString("`UPDATE_SNAPSHOTS=1 go test ./internal/capability/...` para regenerar.\n\n")
	b.WriteString("| Provider | Capability | Description | State | Reason | Test |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, c := range m.Cells {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdownCell(c.Provider),
			escapeMarkdownCell(c.Capability),
			escapeMarkdownCell(c.Description),
			escapeMarkdownCell(string(c.State)),
			escapeMarkdownCell(c.Reason),
			escapeMarkdownCell(c.Test),
		)
	}
	return []byte(b.String())
}

func escapeMarkdownCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
