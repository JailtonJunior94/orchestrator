package contextgen

import (
	"strings"
	"testing"
)

func TestMergeCodexInstallConfigPreservesAuthoredTOMLByteIdentical(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"root key and table":  "model = \"gpt-5\"\n\n[minha_secao]\nchave = \"DELTA\"\n",
		"only table":          "[minha_secao]\nchave = \"DELTA\"\n",
		"only root keys":      "model = \"gpt-5\"\nreasoning = \"high\"\n",
		"no trailing newline": "model = \"gpt-5\"",
	}
	generated := "sandbox_mode = \"workspace-write\"\n\n[[skills.config]]\npath = \".agents/skills/review\"\nenabled = true\n"

	for name, authored := range cases {
		t.Run(name, func(t *testing.T) {
			merged, wasMerged := MergeCodexInstallConfig(generated, authored)
			if !wasMerged {
				t.Fatalf("conteudo autoral deveria ser classificado como merge; got merged=false")
			}
			if !strings.Contains(merged, "[[skills.config]]") {
				t.Fatalf("merge perdeu o bloco gerado:\n%s", merged)
			}
			restored := StripCodexGeneratedRegion(merged)
			if restored != authored {
				t.Fatalf("merge reverso nao e byte-identico:\ngot:  %q\nwant: %q", restored, authored)
			}
		})
	}
}

func TestMergeCodexInstallConfigIsIdempotent(t *testing.T) {
	t.Parallel()
	authored := "model = \"gpt-5\"\n\n[minha_secao]\nchave = \"DELTA\"\n"
	generated := "sandbox_mode = \"workspace-write\"\n\n[[skills.config]]\npath = \".agents/skills/review\"\nenabled = true\n"

	first, _ := MergeCodexInstallConfig(generated, authored)
	second, _ := MergeCodexInstallConfig(generated, first)
	if first != second {
		t.Fatalf("merge nao e idempotente:\nprimeira:\n%s\nsegunda:\n%s", first, second)
	}
	if StripCodexGeneratedRegion(second) != authored {
		t.Fatalf("segunda aplicacao perdeu conteudo autoral: %q", StripCodexGeneratedRegion(second))
	}
}

func TestMergeCodexInstallConfigWithoutAuthoredContentStaysPlain(t *testing.T) {
	t.Parallel()
	generated := "sandbox_mode = \"workspace-write\"\n"
	merged, wasMerged := MergeCodexInstallConfig(generated, "")
	if wasMerged {
		t.Error("arquivo criado do zero nao e merge")
	}
	if merged != generated {
		t.Errorf("conteudo gerado do zero deveria ser byte-identico ao template: %q", merged)
	}
}

func TestMergeCodexInstallConfigDropsLegacyGeneratedContent(t *testing.T) {
	t.Parallel()
	legacy := "model = \"gpt-5\"\n\nsandbox_mode = \"workspace-write\"\napproval_policy = \"on-request\"\n\n[[skills.config]]\npath = \".agents/skills/review\"\nenabled = true\n\n[[hooks.PreToolUse]]\n[[hooks.PreToolUse.hooks]]\ntype = \"command\"\ncommand = \"bash .codex/hooks/validate-preload.sh\"\n"
	generated := "sandbox_mode = \"workspace-write\"\n\n[[skills.config]]\npath = \".agents/skills/bugfix\"\nenabled = true\n"

	merged, _ := MergeCodexInstallConfig(generated, legacy)
	restored := StripCodexGeneratedRegion(merged)
	if strings.Contains(restored, "[[hooks.PreToolUse]]") {
		t.Errorf("conteudo legado do harness deveria ser descartado, nao preservado como autoral:\n%s", restored)
	}
	if !strings.Contains(restored, "model = \"gpt-5\"") {
		t.Errorf("chave autoral perdida na migracao do formato legado:\n%s", restored)
	}
}

func TestMergeCodexInstallConfigUserRootKeyWins(t *testing.T) {
	t.Parallel()
	authored := "sandbox_mode = \"read-only\"\n\n[minha_secao]\nchave = \"DELTA\"\n"
	generated := "sandbox_mode = \"workspace-write\"\n\n[[skills.config]]\npath = \".agents/skills/review\"\nenabled = true\n"

	merged, _ := MergeCodexInstallConfig(generated, authored)
	if strings.Contains(merged, "workspace-write") {
		t.Errorf("chave raiz declarada pelo usuario deve vencer, sem duplicar a chave TOML:\n%s", merged)
	}
	if !strings.Contains(merged, "read-only") {
		t.Errorf("valor do usuario perdido:\n%s", merged)
	}
}
