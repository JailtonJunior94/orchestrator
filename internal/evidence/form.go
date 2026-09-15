package evidence

import (
	"regexp"
	"strings"
)

var accentFolder = strings.NewReplacer(
	"ã", "a", "á", "a", "à", "a", "â", "a",
	"é", "e", "ê", "e",
	"í", "i",
	"ó", "o", "õ", "o", "ô", "o",
	"ú", "u",
	"ç", "c",
)

func fold(raw string) string {
	return accentFolder.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

var (
	formCommandRe   = regexp.MustCompile(`^(?:go\s+(?:test|build|vet|run)\s+\S+|gotestsum\s+\S+|golangci-lint\s+run\b|gofmt\s+\S+|bash\s+\S+|sh\s+\S+|make\s+[A-Za-z0-9][A-Za-z0-9_.\-]*|grep\s+\S+|rg\s+\S+|python3?\s+\S+|pytest\s+\S+|npm\s+(?:run\s+)?\S+|pnpm\s+\S+|yarn\s+\S+|cargo\s+\S+|dotnet\s+\S+|git\s+(?:diff|log|show|status|rev-parse|grep|blame)\b|shasum\s+\S+|sha256sum\s+\S+|\./\S+)`)
	formTestNameRe  = regexp.MustCompile(`^(?:Test|Benchmark|Example)[A-Za-z0-9_/]*$`)
	formTestResult  = regexp.MustCompile(`^(?i:pass|fail)$`)
	formCanonicalRe = regexp.MustCompile(`^(?i:pass(?:ed)?|fail(?:ed)?|exit\s+\d+)$`)
	formTrivialRe   = regexp.MustCompile(`^(?i:ok|okay|done|feito|pronto|sim|yes|no|nao|certo|tudo certo|tudo ok|aprovado|abc|talvez|maybe|n/?a|[-._]+)$`)
	formSignalRe    = regexp.MustCompile(`(?:[0-9]|[A-Za-z0-9_-]+/[A-Za-z0-9_./-]+|[A-Za-z0-9_-]+\.(?:go|py|ts|tsx|js|jsx|cs|rs|java|rb|sh|sql|md|ya?ml|json|toml)\b|(?:Test|Benchmark|Example)[A-Za-z0-9_]+)`)
	formFileLineRe  = regexp.MustCompile(`[A-Za-z0-9_./-]+:[0-9]+`)
)

func substantiveRecord(record string) bool {
	trimmed := strings.TrimSpace(record)
	if trimmed == "" {
		return false
	}
	if formCanonicalRe.MatchString(trimmed) {
		return true
	}
	if formTrivialRe.MatchString(trimmed) {
		return false
	}
	if len(strings.Fields(trimmed)) < 2 {
		return false
	}
	return formSignalRe.MatchString(trimmed)
}

func evidenceFormProblem(evidence string) string {
	value := strings.TrimSpace(evidence)
	if value == "" {
		return "criterio sem linha de evidencia"
	}
	if formFileLineRe.MatchString(value) {
		return ""
	}
	name, record, found := strings.Cut(value, "->")
	if !found {
		return "linha de evidencia fora das tres formas de RF-48 (comando+saida, arquivo:linha, teste+resultado)"
	}
	name = strings.TrimSpace(name)
	record = strings.TrimSpace(record)
	if formTestNameRe.MatchString(name) {
		if !formTestResult.MatchString(record) {
			return "evidencia de teste sem resultado canonico pass/fail (RF-48)"
		}
		return ""
	}
	if !formCommandRe.MatchString(name) {
		return "evidencia nao e comando executavel, referencia arquivo:linha nem resultado de teste (RF-48)"
	}
	if !substantiveRecord(record) {
		return "evidencia nao registra saida significativa do comando — registro trivial nao e prova (RF-48)"
	}
	return ""
}
