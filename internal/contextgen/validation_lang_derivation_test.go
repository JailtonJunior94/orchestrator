package contextgen

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestValidationLangOrderDerivesFromAllLangs(t *testing.T) {
	got := ValidationLangOrder
	if len(got) != len(skills.AllLangs) {
		t.Fatalf("ValidationLangOrder has %d entries, skills.AllLangs has %d; derivation is stale", len(got), len(skills.AllLangs))
	}
	for i, lang := range skills.AllLangs {
		if got[i] != string(lang) {
			t.Errorf("ValidationLangOrder[%d] = %q, want %q (order must match skills.AllLangs)", i, got[i], string(lang))
		}
	}
}

func TestValidationLangLabelsCoverAllLangs(t *testing.T) {
	if len(ValidationLangLabels) != len(skills.AllLangs) {
		t.Fatalf("ValidationLangLabels has %d entries, skills.AllLangs has %d; derivation is stale", len(ValidationLangLabels), len(skills.AllLangs))
	}
	for _, lang := range skills.AllLangs {
		label, ok := ValidationLangLabels[string(lang)]
		if !ok {
			t.Errorf("ValidationLangLabels missing entry for lang %q", lang)
			continue
		}
		if label == "" {
			t.Errorf("ValidationLangLabels[%q] is empty", lang)
		}
	}
}

func TestValidationLangLabelPanicsOnUnknownLang(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("validationLangLabel did not panic for an unmapped lang")
		}
	}()
	validationLangLabel(skills.Lang("cobol"))
}
