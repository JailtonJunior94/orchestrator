package specs

type ProviderLimitation struct {
	Provider    string
	Area        string
	Description string
	Evidence    string
}

var DeclaredProviderLimitations = []ProviderLimitation{
	{
		Provider:    "copilot",
		Area:        "preToolUse-timeout",
		Description: "preToolUse is fail-closed for exit code 2 and for any non-zero exit, but fail-open on hook timeout: a hook that times out does not block the tool call.",
		Evidence:    "measured GitHub Copilot CLI hook contract, recorded in adr-001-hook-contract-fonte-unica-projecoes.md",
	},
	{
		Provider:    "copilot",
		Area:        "trustedFolders-precondition",
		Description: "PreconditionTrustedFolder is declared for Copilot, but the official trustedFolders documentation does not state that it gates hook execution, unlike Codex's trusted hash precondition. The precondition is kept as a conservative declaration with no documentary backing, not a verified gate.",
		Evidence:    "documentation gap identified while researching RF-51 to RF-56",
	},
	{
		Provider:    "all",
		Area:        "after-tool-non-blocking",
		Description: "AfterTool does not block in any of the four CLIs (Claude, Codex, Copilot, OpenCode); the only point with real denial capability is BeforeTool.",
		Evidence:    "measured hook contract across the four CLIs, recorded in adr-001-hook-contract-fonte-unica-projecoes.md",
	},
	{
		Provider:    "all",
		Area:        "permission-request-out-of-scope",
		Description: "PermissionRequest/permissionRequest exists in Claude, Codex and Copilot as a second real denial point and is not used by any adapter in this repository. Registered here as an out-of-scope opportunity for this delivery, not as a gap to fix.",
		Evidence:    "explicitly out of scope for this delivery",
	},
	{
		Provider:    "codex",
		Area:        "hook-trust-silent-skip",
		Description: "Codex silently skips untrusted hooks after a hash change (public bug openai/codex#46210); ai-spec doctor --codex-trust detects this via a read-only hooks/list RPC and is part of the release gate.",
		Evidence:    "openai/codex#46210",
	},
}
