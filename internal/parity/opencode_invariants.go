package parity

import (
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var openCodePermissionHandlerPattern = regexp.MustCompile(`["']permission(\.\w+)?["']\s*:`)

var invOC01OpenCodePluginPresent = &Invariant{
	ID:          "OC01",
	Description: ".opencode/plugin/governance.js instalado, declara o gate tool.execute.before e nunca usa handler de permissao",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolOpenCode},
	Check: func(s Snapshot) Result {
		content := s.File(specs.OpenCodePluginDir + "/governance.js")
		if content == "" {
			return NewChecker().fail(".opencode/plugin/governance.js nao instalado")
		}
		if !strings.Contains(content, "tool.execute.before") {
			return NewChecker().fail(".opencode/plugin/governance.js nao registra o hook 'tool.execute.before' (RF-19)")
		}
		if !strings.Contains(content, "MUTATING_TOOLS") {
			return NewChecker().fail(".opencode/plugin/governance.js nao declara MUTATING_TOOLS")
		}
		if openCodePermissionHandlerPattern.MatchString(content) {
			return NewChecker().fail(".opencode/plugin/governance.js declara handler 'permission'; V-06 provou que permission.ask e codigo morto no OpenCode 1.18.30 e um gate baseado nele falharia em silencio (RF-19)")
		}
		return NewChecker().pass()
	},
}

var invOC02OpenCodeConfigPermissionNeverAsks = &Invariant{
	ID:          "OC02",
	Description: "opencode.json tem bloco permission sem \"ask\" e preserva as ferramentas exigidas",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolOpenCode},
	Check: func(s Snapshot) Result {
		content := s.File(specs.OpenCodeConfigFileName)
		if content == "" {
			return NewChecker().fail("opencode.json nao gerado")
		}
		permission, err := NewChecker().decodeOpenCodePermission(content)
		if err != nil {
			return NewChecker().failf("opencode.json invalido: %v", err)
		}
		if len(permission) == 0 {
			return NewChecker().fail("opencode.json nao declara bloco 'permission' (RF-20)")
		}
		requiredTools := []string{"bash", "edit", "write", "multiedit", "patch"}
		if err := specs.ValidatePermissionBlock(permission, requiredTools...); err != nil {
			return NewChecker().failf("bloco permission de opencode.json invalido: %v", err)
		}
		return NewChecker().pass()
	},
}

var invOC03OpenCodeConfigNeverWritesSkillsOrInstructions = &Invariant{
	ID:          "OC03",
	Description: "opencode.json nao escreve skills.paths nem instructions (RF-13: auto-carregamento nativo)",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolOpenCode},
	Check: func(s Snapshot) Result {
		content := s.File(specs.OpenCodeConfigFileName)
		if content == "" {
			return NewChecker().fail("opencode.json nao gerado")
		}
		doc, err := NewChecker().decodeOpenCodeDocument(content)
		if err != nil {
			return NewChecker().failf("opencode.json invalido: %v", err)
		}
		for _, forbidden := range []string{"skills", "instructions"} {
			if _, found := doc[forbidden]; found {
				return NewChecker().failf("opencode.json declara %q; o OpenCode ja auto-carrega AGENTS.md e .agents/skills/ (RF-13)", forbidden)
			}
		}
		return NewChecker().pass()
	},
}
