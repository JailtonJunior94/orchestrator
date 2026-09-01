package tasks_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd/tasks"
)

func FuzzParserParse(f *testing.F) {
	f.Add([]byte(validPRD), []byte(validTasks))
	f.Add([]byte(""), []byte(""))
	f.Add([]byte("## Requisitos Funcionais\n- RF-01: x\n"), []byte("| # |\n"))
	f.Fuzz(func(t *testing.T, prd, taskFile []byte) {
		_, _ = tasks.NewParser().Parse(prd, taskFile)
	})
}
