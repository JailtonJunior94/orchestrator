package parity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type RP03Suite struct {
	suite.Suite
}

func TestRP03Suite(t *testing.T) {
	suite.Run(t, new(RP03Suite))
}

// TestRP03_CrossCLI_ShellOpNormalizationParity é a prova autoritativa de RP-03: a operação de
// shell, expressa no vocabulário nativo de cada CLI, normaliza para o MESMO normalized_name
// ("bash") e a MESMA chave canônica de input ("command") nas 4 CLIs — usando a normalização de
// produção (events.BuildNormalizedToolCallByDriver). Determinístico, sem rede (R-TEST-001).
func (s *RP03Suite) TestRP03_CrossCLI_ShellOpNormalizationParity() {

	type nativeCall struct {
		driver   string
		rawName  string
		rawInput string
	}
	// Vocabulário nativo de cada CLI para "executar comando shell".
	calls := []nativeCall{
		{"claude", "bash", `{"command":"echo hello"}`},
		{"codex", "shell", `{"cmd":"echo hello"}`},
		{"copilot", "run", `{"command":"echo hello"}`},
		{"gemini", "bash", `{"command":"echo hello"}`},
	}

	const wantName = "bash"
	const wantKey = "command"

	names := map[string]bool{}
	for _, c := range calls {
		drv, err := specs.NewCatalog().ParseDriverID(c.driver)
		if err != nil {
			s.T().Fatalf("ParseDriverID(%q): %v", c.driver, err)
		}
		norm, err := events.NewCatalog().BuildNormalizedToolCallByDriver(drv, c.rawName, json.RawMessage(c.rawInput), "")
		if err != nil {
			s.T().Fatalf("[%s] BuildNormalizedToolCallByDriver: %v", c.driver, err)
		}

		// normalized_name idêntico nas 4 CLIs.
		if norm.NormalizedName != wantName {
			s.T().Errorf("[%s] normalized_name = %q, want %q", c.driver, norm.NormalizedName, wantName)
		}
		names[norm.NormalizedName] = true

		// Forma de input canônica idêntica: exatamente a chave "command".
		var got map[string]json.RawMessage
		if err := json.Unmarshal(norm.NormalizedInput, &got); err != nil {
			s.T().Fatalf("[%s] unmarshal normalized input: %v", c.driver, err)
		}
		if _, ok := got[wantKey]; !ok {
			s.T().Errorf("[%s] input normalizado sem a chave canônica %q; got %v", c.driver, wantKey, keysOfRaw(got))
		}
		if len(got) != 1 {
			s.T().Errorf("[%s] input normalizado deveria ter exatamente 1 chave (%q); got %v", c.driver, wantKey, keysOfRaw(got))
		}
	}

	// O conjunto de normalized_name das 4 CLIs deve ser o singleton {"bash"} (RP-03).
	if len(names) != 1 {
		s.T().Errorf("RP-03: as 4 CLIs deveriam convergir para um único normalized_name; got %v", keysOfBool(names))
	}
}

// TestRP03_RawInputNeverMutated garante que a normalização não muta o RawInput original
// (invariante de paridade: raw preservado byte-a-byte ao lado do normalizado).
func (s *RP03Suite) TestRP03_RawInputNeverMutated() {

	drv, err := specs.NewCatalog().ParseDriverID("codex")
	if err != nil {
		s.T().Fatalf("ParseDriverID: %v", err)
	}
	raw := `{"cmd":"echo hello"}`
	norm, err := events.NewCatalog().BuildNormalizedToolCallByDriver(drv, "shell", json.RawMessage(raw), "")
	if err != nil {
		s.T().Fatalf("BuildNormalizedToolCallByDriver: %v", err)
	}
	if string(norm.RawInput) != raw {
		s.T().Errorf("RawInput mutado: got %q, want %q", string(norm.RawInput), raw)
	}
	if norm.RawName != "shell" {
		s.T().Errorf("RawName mutado: got %q, want %q", norm.RawName, "shell")
	}
}

// ── INV-32 (RP-03 via framework ADR-008) ─────────────────────────────────────

// TestParity_INV32_PassesWhen4CLIsAgree valida que INV-32 passa quando as 4 fixtures
// concordam no normalized_name (mesma operação → mesmo conjunto).
func (s *RP03Suite) TestParity_INV32_PassesWhen4CLIsAgree() {
	snap := snapshotWith4Fixtures("bash", "bash", "bash", "bash")
	r := _invINV32CrossCLIToolCallNameParity.Check(snap)
	if !r.OK {
		s.T().Errorf("INV-32 deveria passar quando as 4 CLIs concordam: %s", r.Reason)
	}
}

// TestParity_INV32_FailsWhenDiverge valida que INV-32 falha quando uma CLI diverge.
func (s *RP03Suite) TestParity_INV32_FailsWhenDiverge() {
	snap := snapshotWith4Fixtures("bash", "bash", "execute", "bash") // copilot diverge
	r := _invINV32CrossCLIToolCallNameParity.Check(snap)
	if r.OK {
		s.T().Error("INV-32 deveria falhar quando o normalized_name diverge entre CLIs")
	}
}

// TestParity_INV32_SkipsWhenFixtureAbsent valida que INV-32 passa (skip) quando falta fixture.
func (s *RP03Suite) TestParity_INV32_SkipsWhenFixtureAbsent() {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolGemini},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			// apenas 2 das 4 fixtures presentes
			testProjectDir + "/tests/fixtures/parity/claude_bash.jsonl": fixtureLine("bash"),
			testProjectDir + "/tests/fixtures/parity/codex_shell.jsonl": fixtureLine("bash"),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := _invINV32CrossCLIToolCallNameParity.Check(snap)
	if !r.OK {
		s.T().Errorf("INV-32 deveria passar (skip) quando alguma fixture está ausente: %s", r.Reason)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func fixtureLine(normalizedName string) []byte {
	return []byte(`{"kind":"tool_call_start","raw_name":"x","normalized_name":"` + normalizedName + `"}` + "\n")
}

func snapshotWith4Fixtures(claude, codex, copilot, gemini string) Snapshot {
	return Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolGemini},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/tests/fixtures/parity/claude_bash.jsonl": fixtureLine(claude),
			testProjectDir + "/tests/fixtures/parity/codex_shell.jsonl": fixtureLine(codex),
			testProjectDir + "/tests/fixtures/parity/copilot_run.jsonl": fixtureLine(copilot),
			testProjectDir + "/tests/fixtures/parity/gemini_bash.jsonl": fixtureLine(gemini),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
}

func keysOfRaw(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func keysOfBool(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
