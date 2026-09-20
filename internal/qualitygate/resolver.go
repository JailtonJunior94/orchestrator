package qualitygate

import (
	"errors"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type ResolvedCheck struct {
	Kind     CheckKind
	Command  string
	Required bool
}

var ErrCheckUnavailableForStack = errors.New("qualitygate: required check has no resolved command for detected stack")

var ErrNoStackDetected = errors.New("qualitygate: no stack detected in toolchain result")

func ResolveChecks(selection Selection, toolchain detect.ToolchainResult, lang string) ([]ResolvedCheck, error) {
	entry, ok := toolchain[lang]
	if !ok {
		return nil, fmt.Errorf("%w: lang=%s", ErrNoStackDetected, lang)
	}

	resolved := make([]ResolvedCheck, 0, len(selection.required)+len(selection.optional))

	for _, kind := range selection.Required() {
		command := commandFor(entry, kind)
		if command == "" {
			return nil, fmt.Errorf("%w: check=%s lang=%s", ErrCheckUnavailableForStack, kind, lang)
		}
		resolved = append(resolved, ResolvedCheck{Kind: kind, Command: command, Required: true})
	}

	for _, kind := range selection.Optional() {
		command := commandFor(entry, kind)
		if command == "" {
			continue
		}
		resolved = append(resolved, ResolvedCheck{Kind: kind, Command: command, Required: false})
	}

	return resolved, nil
}

func commandFor(entry detect.ToolchainEntry, kind CheckKind) string {
	switch kind {
	case CheckFmt:
		return entry.Fmt
	case CheckTest:
		return entry.Test
	case CheckLint:
		return entry.Lint
	default:
		panic(fmt.Sprintf("qualitygate: unhandled check kind %v", kind))
	}
}

func PrimaryLang(toolchain detect.ToolchainResult) (string, bool) {
	for _, lang := range skills.AllLangs {
		if _, ok := toolchain[string(lang)]; ok {
			return string(lang), true
		}
	}
	for lang := range toolchain {
		return lang, true
	}
	return "", false
}
