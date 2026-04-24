package taskloop

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestClassifyIterationOutcome verifica todas as combinacoes de input da funcao pura
// classifyIterationOutcome conforme as regras definidas na techspec (D3) e task 7.0.
func TestClassifyIterationOutcome(t *testing.T) {
	errInvoke := errors.New("dial tcp: connection refused")

	tests := []struct {
		name        string
		preStatus   string
		postStatus  string
		exitCode    int
		invokeErr   error
		stdout      string
		stderr      string
		wantSkip    bool
		wantAbort   bool
		wantReview  bool
		noteContains []string
	}{
		// ---- invokeErr != nil ----
		{
			name:         "invokeErr: skip, sem revisao",
			preStatus:    "pending",
			postStatus:   "pending",
			exitCode:     0,
			invokeErr:    errInvoke,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"erro de invocacao", "connection refused"},
		},
		{
			name:         "invokeErr com postStatus done: skip prevalece, sem revisao",
			preStatus:    "pending",
			postStatus:   "done",
			exitCode:     0,
			invokeErr:    errInvoke,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"erro de invocacao"},
		},
		{
			name:         "invokeErr com postStatus inalterado: skip + status inalterado acumulado",
			preStatus:    "pending",
			postStatus:   "pending",
			exitCode:     0,
			invokeErr:    errInvoke,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"erro de invocacao", "status inalterado"},
		},

		// ---- isAuthError (exitCode != 0) ----
		{
			name:        "auth error em stdout: abort",
			preStatus:   "pending",
			postStatus:  "pending",
			exitCode:    1,
			invokeErr:   nil,
			stdout:      "Error: not authenticated — please run /login",
			wantSkip:    false,
			wantAbort:   true,
			wantReview:  false,
			noteContains: []string{"erro de autenticacao"},
		},
		{
			name:        "auth error em stderr: abort",
			preStatus:   "pending",
			postStatus:  "pending",
			exitCode:    1,
			invokeErr:   nil,
			stderr:      "unauthorized access",
			wantSkip:    false,
			wantAbort:   true,
			wantReview:  false,
			noteContains: []string{"erro de autenticacao"},
		},
		{
			name:        "auth error com exitCode zero: nao detectado (isAuthError so verifica exitCode!=0)",
			preStatus:   "pending",
			postStatus:  "pending",
			exitCode:    0,
			invokeErr:   nil,
			stdout:      "not authenticated",
			wantSkip:    true,
			wantAbort:   false,
			wantReview:  false,
			noteContains: []string{"status inalterado"},
		},

		// ---- exitCode != 0, sem auth error ----
		{
			name:         "exitCode != 0 sem auth, postStatus inalterado: note + skip",
			preStatus:    "pending",
			postStatus:   "pending",
			exitCode:     1,
			invokeErr:    nil,
			stdout:       "build failed",
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"agente saiu com codigo 1", "status inalterado"},
		},
		{
			name:         "exitCode != 0 sem auth, postStatus done: note + runReviewer",
			preStatus:    "pending",
			postStatus:   "done",
			exitCode:     1,
			invokeErr:    nil,
			wantSkip:     false,
			wantAbort:    false,
			wantReview:   true,
			noteContains: []string{"agente saiu com codigo 1"},
		},
		{
			name:         "exitCode -1 (timeout), postStatus done: runReviewer (exit code irrelevante)",
			preStatus:    "pending",
			postStatus:   "done",
			exitCode:     -1,
			invokeErr:    nil,
			wantSkip:     false,
			wantAbort:    false,
			wantReview:   true,
			noteContains: []string{"agente saiu com codigo -1"},
		},

		// ---- sucesso: exit 0, postStatus done ----
		{
			name:         "sucesso: exit 0, done, sem nota de erro",
			preStatus:    "pending",
			postStatus:   "done",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     false,
			wantAbort:    false,
			wantReview:   true,
			noteContains: []string{},
		},

		// ---- postStatus == preStatus ----
		{
			name:         "status inalterado: exit 0, skip",
			preStatus:    "pending",
			postStatus:   "pending",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"status inalterado"},
		},
		{
			name:         "status inalterado in_progress: exit 0, skip",
			preStatus:    "in_progress",
			postStatus:   "in_progress",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"status inalterado"},
		},

		// ---- postStatus terminal nao-done ----
		{
			name:         "postStatus failed: skip, sem revisao",
			preStatus:    "pending",
			postStatus:   "failed",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{},
		},
		{
			name:         "postStatus blocked: skip",
			preStatus:    "pending",
			postStatus:   "blocked",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{},
		},
		{
			name:         "postStatus needs_input: skip",
			preStatus:    "pending",
			postStatus:   "needs_input",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{},
		},
		{
			name:         "postStatus failed com exitCode != 0: note + skip",
			preStatus:    "pending",
			postStatus:   "failed",
			exitCode:     2,
			invokeErr:    nil,
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"agente saiu com codigo 2"},
		},

		// ---- postStatus muda de pending para in_progress (avanco parcial) ----
		{
			name:         "postStatus in_progress: sem skip, sem revisao",
			preStatus:    "pending",
			postStatus:   "in_progress",
			exitCode:     0,
			invokeErr:    nil,
			wantSkip:     false,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{},
		},

		// ---- note de exitCode nao inclui ferramenta (responsabilidade do caller) ----
		{
			name:         "exitCode -1 saida vazia: note basica sem nome de ferramenta",
			preStatus:    "pending",
			postStatus:   "pending",
			exitCode:     -1,
			invokeErr:    nil,
			stdout:       "",
			stderr:       "",
			wantSkip:     true,
			wantAbort:    false,
			wantReview:   false,
			noteContains: []string{"agente saiu com codigo -1", "status inalterado"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyIterationOutcome(
				tt.preStatus,
				tt.postStatus,
				tt.exitCode,
				tt.invokeErr,
				tt.stdout,
				tt.stderr,
			)

			if got.Skip != tt.wantSkip {
				t.Errorf("Skip = %v, want %v", got.Skip, tt.wantSkip)
			}
			if got.Abort != tt.wantAbort {
				t.Errorf("Abort = %v, want %v", got.Abort, tt.wantAbort)
			}
			if got.RunReviewer != tt.wantReview {
				t.Errorf("RunReviewer = %v, want %v", got.RunReviewer, tt.wantReview)
			}
			for _, fragment := range tt.noteContains {
				if !strings.Contains(got.Note, fragment) {
					t.Errorf("Note = %q; esperava conter %q", got.Note, fragment)
				}
			}
		})
	}
}

// TestClassifyIterationOutcomeNoteFormat verifica o formato exato das notas
// para os cenarios mais criticos (regressao de texto observavel).
func TestClassifyIterationOutcomeNoteFormat(t *testing.T) {
	tests := []struct {
		name       string
		preStatus  string
		postStatus string
		exitCode   int
		invokeErr  error
		stdout     string
		stderr     string
		wantNote   string
	}{
		{
			name:       "exit code nota exata",
			preStatus:  "pending",
			postStatus: "in_progress",
			exitCode:   3,
			invokeErr:  nil,
			wantNote:   "agente saiu com codigo 3",
		},
		{
			name:       "auth error nota exata",
			preStatus:  "pending",
			postStatus: "pending",
			exitCode:   1,
			invokeErr:  nil,
			stdout:     "api key invalid",
			wantNote:   "erro de autenticacao",
		},
		{
			name:       "status inalterado nota exata",
			preStatus:  "pending",
			postStatus: "pending",
			exitCode:   0,
			invokeErr:  nil,
			wantNote:   "status inalterado apos execucao; pulando",
		},
		{
			name:       "invokeErr nota contem mensagem de erro",
			preStatus:  "pending",
			postStatus: "pending",
			exitCode:   0,
			invokeErr:  fmt.Errorf("contexto: %w", errors.New("causa")),
			wantNote:   "erro de invocacao: contexto: causa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyIterationOutcome(
				tt.preStatus, tt.postStatus,
				tt.exitCode, tt.invokeErr,
				tt.stdout, tt.stderr,
			)

			if !strings.Contains(got.Note, tt.wantNote) {
				t.Errorf("Note = %q; esperava conter %q", got.Note, tt.wantNote)
			}
		})
	}
}
