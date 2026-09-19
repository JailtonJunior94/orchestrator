package conformance

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func repoRootForConformance(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root walking up from %s", dir)
		}
		dir = parent
	}
}

type prdScenarioRow struct {
	id          int
	name        string
	determinist bool
	live        bool
	decision    string
}

func parsePRDClassificationTable(t *testing.T, prdPath string) []prdScenarioRow {
	t.Helper()
	data, err := os.ReadFile(prdPath)
	if err != nil {
		t.Fatalf("read prd.md: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	heading := "## Classificação dos Cenários de Conformidade"
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i
			break
		}
	}
	if start == -1 {
		t.Fatalf("prd.md must declare the section %q", heading)
	}

	var rows []prdScenarioRow
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			if len(rows) > 0 {
				break
			}
			continue
		}
		fields := splitTableRow(trimmed)
		if len(fields) != 5 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		rows = append(rows, prdScenarioRow{
			id:          id,
			name:        fields[1],
			determinist: fields[2] == "✓",
			live:        fields[3] == "✓",
			decision:    fields[4],
		})
	}
	return rows
}

func splitTableRow(row string) []string {
	trimmed := strings.Trim(row, "|")
	parts := strings.Split(trimmed, "|")
	fields := make([]string, 0, len(parts))
	for _, p := range parts {
		fields = append(fields, strings.TrimSpace(p))
	}
	return fields
}

func diffManifestAgainstPRDRows(manifest []Scenario, prdRows []prdScenarioRow) []string {
	var problems []string
	if len(prdRows) != len(manifest) {
		problems = append(problems, "row count diverges between prd.md and the manifest")
	}

	byID := make(map[int]Scenario, len(manifest))
	for _, s := range manifest {
		byID[s.ID] = s
	}

	for _, row := range prdRows {
		scenario, ok := byID[row.id]
		if !ok {
			problems = append(problems, "prd.md declares a scenario not present in the manifest")
			continue
		}
		if scenario.Name != row.name {
			problems = append(problems, "scenario name diverges")
		}
		if scenario.Determinist != row.determinist {
			problems = append(problems, "scenario determinist classification diverges")
		}
		if scenario.Live != row.live {
			problems = append(problems, "scenario live classification diverges")
		}
		if scenario.Decision != row.decision {
			problems = append(problems, "scenario decision text diverges")
		}
	}
	return problems
}

func TestManifestReplicatesPRDNormativeClassification(t *testing.T) {
	root := repoRootForConformance(t)
	prdPath := filepath.Join(root, ".specs", "prd-harness-portatil-vendor-neutral", "prd.md")
	prdRows := parsePRDClassificationTable(t, prdPath)

	if problems := diffManifestAgainstPRDRows(Manifest(), prdRows); len(problems) > 0 {
		t.Fatalf("manifest diverges from the prd.md normative classification table: %v", problems)
	}
}

func TestManifestDivergenceFromPRDIsDetected(t *testing.T) {
	prdRows := []prdScenarioRow{
		{id: 1, name: "Tarefa simples de leitura", determinist: false, live: true, decision: "Depende da resposta do modelo; nada a decidir por contrato"},
	}
	mutated := []Scenario{
		{ID: 1, Name: "Tarefa simples de leitura", Determinist: true, Live: true, Decision: "Depende da resposta do modelo; nada a decidir por contrato"},
	}
	if problems := diffManifestAgainstPRDRows(mutated, prdRows); len(problems) == 0 {
		t.Fatal("a manifest whose determinist classification diverges from prd.md must be detected as a divergence, not pass silently")
	}
}

func TestManifestDeterministicScenariosAreAllRunnableByFixture(t *testing.T) {
	for _, s := range Manifest() {
		if !s.Determinist {
			continue
		}
		if s.Decision == "" {
			t.Errorf("scenario #%d (%q) is classified as deterministic but declares no fixture/contract decision rule", s.ID, s.Name)
		}
	}
}

func TestScenariosWithDualClassificationMatchPRDDecision(t *testing.T) {
	dualIDs := map[int]bool{6: true, 7: true, 8: true}
	for _, s := range Manifest() {
		if dualIDs[s.ID] {
			if !s.Determinist || !s.Live {
				t.Errorf("scenario #%d (%q) must be classified as both deterministic and live per PRD decision D-14", s.ID, s.Name)
			}
		}
	}
}
