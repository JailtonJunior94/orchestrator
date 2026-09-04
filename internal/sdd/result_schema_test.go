package sdd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func TestResultValidatorValidateExecutionJSON(t *testing.T) {
	valid := []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}`)
	valid = []byte(strings.Replace(string(valid), `"final_state_sha256"`, `"patch_ref":"patch.diff","final_state_sha256"`, 1))
	extra := append(append([]byte(nil), valid[:len(valid)-1]...), []byte(`,"extra":true}`)...)
	cases := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{name: "aceita resultado completo", content: valid},
		{name: "rejeita campo desconhecido", content: extra, wantErr: true},
		{name: "rejeita done sem revisao aprovada", content: []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"changes_requested"}`), wantErr: true},
		{name: "rejeita evidencia com traversal", content: []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"../../segredo.md"}],"evidence":["report.md"],"review_verdict":"approved"}`), wantErr: true},
		{name: "rejeita evidencia absoluta Windows", content: []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["C:\\\\temp\\\\segredo.md"],"review_verdict":"approved"}`), wantErr: true},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := sdd.NewResultValidator().ValidateExecutionJSON(scenario.content)
			if (err != nil) != scenario.wantErr {
				t.Fatalf("erro=%v, wantErr=%t", err, scenario.wantErr)
			}
		})
	}
}

func TestResultValidatorRejectsAbsoluteEvidencePathsOnEveryPlatform(t *testing.T) {
	valid := `{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}`
	valid = strings.Replace(valid, `"final_state_sha256"`, `"patch_ref":"patch.diff","final_state_sha256"`, 1)
	cases := []struct {
		name      string
		reference string
		wantErr   bool
	}{
		{name: "aceita caminho relativo POSIX", reference: "evidence/report.md"},
		{name: "aceita caminho relativo Windows", reference: `evidence\report.md`},
		{name: "rejeita caminho absoluto POSIX", reference: "/etc/passwd", wantErr: true},
		{name: "rejeita caminho com raiz Windows", reference: `\Windows\System32\config`, wantErr: true},
		{name: "rejeita caminho com volume Windows", reference: `C:\evidence\report.md`, wantErr: true},
		{name: "rejeita caminho UNC Windows", reference: `\\servidor\compartilhamento\report.md`, wantErr: true},
		{name: "rejeita caminho Windows com prefixo estendido", reference: `\\?\C:\evidence\report.md`, wantErr: true},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			referenceJSON, err := json.Marshal(scenario.reference)
			if err != nil {
				t.Fatalf("serializar referencia: %v", err)
			}
			content := []byte(strings.ReplaceAll(valid, `"report.md"`, string(referenceJSON)))
			_, err = sdd.NewResultValidator().ValidateExecutionJSON(content)
			if (err != nil) != scenario.wantErr {
				t.Fatalf("erro=%v, wantErr=%t", err, scenario.wantErr)
			}
			if scenario.wantErr && !strings.Contains(err.Error(), "referencia de evidencia deve ser caminho relativo") {
				t.Fatalf("erro deveria rejeitar path absoluto, recebeu: %v", err)
			}
		})
	}
}

func TestResultValidatorValidateReviewJSON(t *testing.T) {
	content := []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"verdict":"approved"}`)
	if _, err := sdd.NewResultValidator().ValidateReviewJSON(content); err != nil {
		t.Fatalf("resultado de revisao valido deveria passar: %v", err)
	}
}

func TestResultValidatorValidateCheckpointJSON(t *testing.T) {
	content := []byte(`{"schema_version":2,"run_id":"run-1","task_id":"2.0","attempt":1,"status":"blocked","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":1,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"needs_input"}`)
	if _, err := sdd.NewResultValidator().ValidateCheckpointJSON(content); err != nil {
		t.Fatalf("checkpoint completo deveria passar: %v", err)
	}
}
