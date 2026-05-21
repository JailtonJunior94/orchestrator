package agents

import "testing"

// T-10: CLI --model setado, AGENT.md tem model diferente -> RuntimeOverride.Model preserva valor da CLI.
func TestApplyRuntimePrecedence_CLIWins(t *testing.T) {
	cfg := &RuntimeOverride{
		Model:         "cli-model",
		ExplicitModel: true,
	}
	defaults := RuntimeDefaults{
		IDE:             "claude",
		Model:           "agent-model",
		ReasoningEffort: "high",
		AccessMode:      "bypass-permissions",
	}

	applyRuntimePrecedence(cfg, defaults)

	// CLI model deve ser preservado.
	if cfg.Model != "cli-model" {
		t.Errorf("T-10: Model = %q, quero %q", cfg.Model, "cli-model")
	}
	// Campos nao explicitados devem ser preenchidos pelos defaults do agente.
	if cfg.IDE != "claude" {
		t.Errorf("T-10: IDE = %q, quero %q", cfg.IDE, "claude")
	}
	if cfg.ReasoningEffort != "high" {
		t.Errorf("T-10: ReasoningEffort = %q, quero %q", cfg.ReasoningEffort, "high")
	}
	if cfg.AccessMode != "bypass-permissions" {
		t.Errorf("T-10: AccessMode = %q, quero %q", cfg.AccessMode, "bypass-permissions")
	}
}

// T-11: CLI sem --model, AGENT.md tem model -> RuntimeOverride.Model = AGENT.md.
func TestApplyRuntimePrecedence_AgentFills(t *testing.T) {
	cfg := &RuntimeOverride{}
	defaults := RuntimeDefaults{
		IDE:   "codex",
		Model: "agent-model",
	}

	applyRuntimePrecedence(cfg, defaults)

	if cfg.IDE != "codex" {
		t.Errorf("T-11: IDE = %q, quero %q", cfg.IDE, "codex")
	}
	if cfg.Model != "agent-model" {
		t.Errorf("T-11: Model = %q, quero %q", cfg.Model, "agent-model")
	}
}

// T-12: CLI sem flags, AGENT.md sem runtime -> RuntimeOverride permanece vazio.
func TestApplyRuntimePrecedence_EmptyBothSides(t *testing.T) {
	cfg := &RuntimeOverride{}
	defaults := RuntimeDefaults{}

	applyRuntimePrecedence(cfg, defaults)

	if cfg.IDE != "" {
		t.Errorf("T-12: IDE deveria ser vazio, got %q", cfg.IDE)
	}
	if cfg.Model != "" {
		t.Errorf("T-12: Model deveria ser vazio, got %q", cfg.Model)
	}
	if cfg.ReasoningEffort != "" {
		t.Errorf("T-12: ReasoningEffort deveria ser vazio, got %q", cfg.ReasoningEffort)
	}
	if cfg.AccessMode != "" {
		t.Errorf("T-12: AccessMode deveria ser vazio, got %q", cfg.AccessMode)
	}
}

// Extra: CLI com todos os campos explicitados -> nenhum default do agente e aplicado.
func TestApplyRuntimePrecedence_AllCLIExplicit(t *testing.T) {
	cfg := &RuntimeOverride{
		IDE:                     "gemini",
		Model:                   "gemini-pro",
		ReasoningEffort:         "low",
		AccessMode:              "readonly",
		ExplicitIDE:             true,
		ExplicitModel:           true,
		ExplicitReasoningEffort: true,
		ExplicitAccessMode:      true,
	}
	defaults := RuntimeDefaults{
		IDE:             "claude",
		Model:           "claude-opus-4-7",
		ReasoningEffort: "high",
		AccessMode:      "bypass-permissions",
	}

	applyRuntimePrecedence(cfg, defaults)

	// Todos os valores CLI devem ser preservados.
	if cfg.IDE != "gemini" {
		t.Errorf("IDE = %q, quero %q", cfg.IDE, "gemini")
	}
	if cfg.Model != "gemini-pro" {
		t.Errorf("Model = %q, quero %q", cfg.Model, "gemini-pro")
	}
	if cfg.ReasoningEffort != "low" {
		t.Errorf("ReasoningEffort = %q, quero %q", cfg.ReasoningEffort, "low")
	}
	if cfg.AccessMode != "readonly" {
		t.Errorf("AccessMode = %q, quero %q", cfg.AccessMode, "readonly")
	}
}

// Extra: CLI com alguns campos explicitados, agente preenche os restantes.
func TestApplyRuntimePrecedence_PartialCLI(t *testing.T) {
	cfg := &RuntimeOverride{
		IDE:         "copilot",
		ExplicitIDE: true,
	}
	defaults := RuntimeDefaults{
		IDE:             "claude",
		Model:           "claude-opus-4-7",
		ReasoningEffort: "medium",
	}

	applyRuntimePrecedence(cfg, defaults)

	// IDE explicitado na CLI deve ser preservado.
	if cfg.IDE != "copilot" {
		t.Errorf("IDE = %q, quero %q", cfg.IDE, "copilot")
	}
	// Model e ReasoningEffort devem vir do agente.
	if cfg.Model != "claude-opus-4-7" {
		t.Errorf("Model = %q, quero %q", cfg.Model, "claude-opus-4-7")
	}
	if cfg.ReasoningEffort != "medium" {
		t.Errorf("ReasoningEffort = %q, quero %q", cfg.ReasoningEffort, "medium")
	}
}
