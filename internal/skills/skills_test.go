package skills

import "testing"

func TestParseTool(t *testing.T) {
	for _, name := range []string{"claude", "gemini", "codex", "copilot"} {
		tool, ok := ParseTool(name)
		if !ok {
			t.Errorf("ParseTool(%q) failed", name)
		}
		if string(tool) != name {
			t.Errorf("ParseTool(%q) = %q", name, tool)
		}
	}
	_, ok := ParseTool("invalid")
	if ok {
		t.Error("ParseTool(invalid) should fail")
	}
}

func TestParseLang(t *testing.T) {
	for _, name := range []string{"go", "node", "python"} {
		lang, ok := ParseLang(name)
		if !ok {
			t.Errorf("ParseLang(%q) failed", name)
		}
		if string(lang) != name {
			t.Errorf("ParseLang(%q) = %q", name, lang)
		}
	}
	_, ok := ParseLang("rust")
	if ok {
		t.Error("ParseLang(rust) should fail")
	}
}

func TestLangSkills(t *testing.T) {
	s := LangSkills([]Lang{LangGo})
	if len(s) != 2 || s[0] != "go-implementation" || s[1] != "object-calisthenics-go" {
		t.Errorf("LangSkills(go) = %v", s)
	}

	s = LangSkills([]Lang{LangNode, LangPython})
	if len(s) != 2 {
		t.Errorf("LangSkills(node, python) = %v", s)
	}
}

func TestAllSkills(t *testing.T) {
	all := AllSkills([]Lang{LangGo})
	want := len(BaseSkills) + len(ComplementarySkills) + 2
	if len(all) != want {
		t.Errorf("AllSkills(go) = %d skills, want %d", len(all), want)
	}
}

func TestComplementarySkills(t *testing.T) {
	if len(ComplementarySkills) != 11 {
		t.Errorf("ComplementarySkills count = %d, want 11", len(ComplementarySkills))
	}
}
