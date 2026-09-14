package parity

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type OpenCodeParitySuite struct {
	suite.Suite
}

func TestOpenCodeParitySuite(t *testing.T) {
	suite.Run(t, new(OpenCodeParitySuite))
}

func (s *OpenCodeParitySuite) openCodeSnapshot() Snapshot {
	snap, err := NewChecker().Generate(testProjectDir, []skills.Tool{skills.ToolOpenCode}, nil, "full")
	s.Require().NoError(err)
	return snap
}

func (s *OpenCodeParitySuite) withFile(snap Snapshot, rel string, content []byte) Snapshot {
	files := cloneFiles(snap.Files)
	files[filepath.Join(snap.ProjectDir, rel)] = content
	snap.Files = files
	return snap
}

func (s *OpenCodeParitySuite) withoutFile(snap Snapshot, rel string) Snapshot {
	files := cloneFiles(snap.Files)
	delete(files, filepath.Join(snap.ProjectDir, rel))
	snap.Files = files
	return snap
}

func (s *OpenCodeParitySuite) TestOpenCodeHasNonVacuousInvariantCoverage() {
	snap := s.openCodeSnapshot()
	results := NewChecker().Run(snap, NewChecker().Invariants())

	toolSpecific := 0
	executed := 0
	for _, cr := range results {
		if cr.Skipped {
			continue
		}
		executed++
		if cr.Invariant.Level != ToolSpecific {
			continue
		}
		for _, tool := range cr.Invariant.AppliesTo {
			if tool == skills.ToolOpenCode {
				toolSpecific++
				break
			}
		}
	}

	s.Positive(executed, "nenhum invariante executou para o agente OpenCode")
	s.GreaterOrEqual(toolSpecific, 3,
		"o agente OpenCode precisa de invariantes ToolSpecific proprios; sem eles a paridade e nominal")
}

func (s *OpenCodeParitySuite) TestOC01FailsWhenPluginMissing() {
	snap := s.withoutFile(s.openCodeSnapshot(), specs.OpenCodePluginDir+"/governance.js")
	s.False(invOC01OpenCodePluginPresent.Check(snap).OK,
		"OC01 deveria falhar quando o plugin de governanca nao esta instalado")
}

func (s *OpenCodeParitySuite) TestOC01FailsWhenPluginDropsTheGate() {
	snap := s.openCodeSnapshot()
	stripped := strings.ReplaceAll(snap.File(specs.OpenCodePluginDir+"/governance.js"),
		"tool.execute.before", "tool.execute.never")
	snap = s.withFile(snap, specs.OpenCodePluginDir+"/governance.js", []byte(stripped))
	s.False(invOC01OpenCodePluginPresent.Check(snap).OK,
		"OC01 deveria falhar quando o plugin nao registra tool.execute.before")
}

func (s *OpenCodeParitySuite) TestOC01PassesOnRealPlugin() {
	s.True(invOC01OpenCodePluginPresent.Check(s.openCodeSnapshot()).OK)
}

func (s *OpenCodeParitySuite) TestOC02FailsWhenPermissionAsks() {
	snap := s.withFile(s.openCodeSnapshot(), specs.OpenCodeConfigFileName,
		[]byte(`{"permission":{"bash":"ask"}}`))
	s.False(invOC02OpenCodeConfigPermissionNeverAsks.Check(snap).OK,
		"OC02 deveria falhar com \"ask\": V-06 provou que permission.ask e codigo morto (RF-20)")
}

func (s *OpenCodeParitySuite) TestOC02FailsWhenPermissionBlockAbsent() {
	snap := s.withFile(s.openCodeSnapshot(), specs.OpenCodeConfigFileName, []byte(`{"model":"x"}`))
	s.False(invOC02OpenCodeConfigPermissionNeverAsks.Check(snap).OK,
		"OC02 deveria falhar quando opencode.json nao declara bloco permission")
}

func (s *OpenCodeParitySuite) TestOC02FailsWhenRequiredToolIsDeniedWholesale() {
	snap := s.withFile(s.openCodeSnapshot(), specs.OpenCodeConfigFileName,
		[]byte(`{"permission":{"bash":"deny"}}`))
	s.False(invOC02OpenCodeConfigPermissionNeverAsks.Check(snap).OK,
		"OC02 deveria falhar quando um deny geral remove a ferramenta do tool-set (V-07)")
}

func (s *OpenCodeParitySuite) TestOC02PassesOnCanonicalConfig() {
	s.True(invOC02OpenCodeConfigPermissionNeverAsks.Check(s.openCodeSnapshot()).OK)
}

func (s *OpenCodeParitySuite) TestOC03FailsWhenConfigWritesSkillsOrInstructions() {
	scenarios := []string{
		`{"permission":{"bash":{"rm -rf*":"deny"}},"skills":{"paths":[".agents/skills"]}}`,
		`{"permission":{"bash":{"rm -rf*":"deny"}},"instructions":["AGENTS.md"]}`,
	}
	for _, content := range scenarios {
		snap := s.withFile(s.openCodeSnapshot(), specs.OpenCodeConfigFileName, []byte(content))
		s.False(invOC03OpenCodeConfigNeverWritesSkillsOrInstructions.Check(snap).OK,
			"OC03 deveria falhar: V-09/V-10 provaram que escrever skills.paths/instructions e redundante e nocivo (RF-13)")
	}
}

func (s *OpenCodeParitySuite) TestOC03PassesOnCanonicalConfig() {
	s.True(invOC03OpenCodeConfigNeverWritesSkillsOrInstructions.Check(s.openCodeSnapshot()).OK)
}

func (s *OpenCodeParitySuite) TestX01CoversOpenCodeThroughAgentsMD() {
	snap := s.openCodeSnapshot()
	s.True(invX01CrossToolCanonicalPath.Check(snap).OK)

	broken := s.withFile(snap, "AGENTS.md", []byte("# sem referencia canonica\n"))
	s.False(invX01CrossToolCanonicalPath.Check(broken).OK,
		"X01 deveria falhar quando o artefato canonico do OpenCode perde '.agents/skills/'")

	absent := s.withoutFile(snap, "AGENTS.md")
	s.False(invX01CrossToolCanonicalPath.Check(absent).OK,
		"X01 deveria falhar quando o artefato canonico do OpenCode esta ausente")
}

func (s *OpenCodeParitySuite) TestINV32ReadsTheOpenCodeFixture() {
	snap := snapshotWith4Fixtures("bash", "bash", "bash", "grep")
	s.False(invINV32CrossCLIToolCallNameParity.Check(snap).OK,
		"INV-32 deveria falhar quando o OpenCode diverge das demais CLIs")

	s.True(invINV32CrossCLIToolCallNameParity.Check(snapshotWith4Fixtures("bash", "bash", "bash", "bash")).OK)
}

func (s *OpenCodeParitySuite) TestOC01FailsWhenPluginReintroducesPermissionHandler() {
	snap := s.openCodeSnapshot()
	withPermission := strings.Replace(snap.File(specs.OpenCodePluginDir+"/governance.js"),
		`"tool.execute.before":`,
		`"permission.ask": async () => ({ status: "deny" }),
    "tool.execute.before":`, 1)
	snap = s.withFile(snap, specs.OpenCodePluginDir+"/governance.js", []byte(withPermission))
	s.False(invOC01OpenCodePluginPresent.Check(snap).OK,
		"OC01 deveria falhar quando o plugin volta a basear o gate em 'permission': V-06 provou que permission.ask e codigo morto (RF-19/V-06)")
}
