package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// tokensPerRefLoad e a estimativa de tokens por carga de arquivo de referencia.
// Baseada em observacao empirica de referencias tipicas (~2000 chars / 3.5).
// ESTIMATIVA: nao representa tokens reais do provedor. Use 'ai-spec-harness metrics' para medicao completa.
const tokensPerRefLoad = 570

// Summary le .agents/telemetry.log, filtra por timestamp >= now-since (zero = sem filtro),
// conta por skill e por ref, exibe custo estimado em tres eixos e retorna string formatada.
func Summary(rootDir string, since time.Duration) (string, error) {
	logPath := filepath.Join(rootDir, ".agents", "telemetry.log")
	if _, statErr := os.Stat(logPath); os.IsNotExist(statErr) {
		return "Sem dados de telemetria.", nil
	}

	entries, err := parseLogEntries(logPath, since)
	if err != nil {
		return "", err
	}

	skillCounts := make(map[string]int)
	refCounts := make(map[string]int)

	for _, e := range entries {
		if e.Skill != "" {
			skillCounts[e.Skill]++
		}
		if e.Ref != "" {
			refCounts[e.Ref]++
		}
	}

	totalLines := len(entries)

	if totalLines == 0 {
		return "Sem dados de telemetria no periodo especificado.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Telemetria (%d entradas)\n\n", totalLines)

	fmt.Fprintf(&sb, "Skills:\n")
	skillKeys := sortedKeys(skillCounts)
	for _, skill := range skillKeys {
		fmt.Fprintf(&sb, "  %s: %d\n", skill, skillCounts[skill])
	}

	fmt.Fprintf(&sb, "\nReferencias:\n")
	refKeys := sortedKeys(refCounts)
	for _, ref := range refKeys {
		fmt.Fprintf(&sb, "  %s: %d\n", ref, refCounts[ref])
	}

	// Custo em tres eixos a partir do log de telemetria.
	// on-disk nao esta disponivel sem scan do filesystem (use 'ai-spec-harness metrics').
	// loaded = refs unicas carregadas nesta janela × custo estimado por ref.
	// incremental-ref = total de loads (incluindo repeticoes) × custo estimado por ref.
	uniqueRefsLoaded := len(refCounts)
	totalRefLoads := 0
	for _, cnt := range refCounts {
		totalRefLoads += cnt
	}
	fmt.Fprintf(&sb, "\nCusto Estimado (ESTIMATIVA operacional — chars/3.5, nao tokens reais do provedor):\n")
	fmt.Fprintf(&sb, "  on-disk        : indisponivel no log — use 'ai-spec-harness metrics'\n")
	fmt.Fprintf(&sb, "  loaded         : %d refs unicas x ~%d tokens = %d tokens est.\n",
		uniqueRefsLoaded, tokensPerRefLoad, uniqueRefsLoaded*tokensPerRefLoad)
	fmt.Fprintf(&sb, "  incremental-ref: %d loads totais x ~%d tokens = %d tokens est.\n",
		totalRefLoads, tokensPerRefLoad, totalRefLoads*tokensPerRefLoad)

	return sb.String(), nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
