package taskloop

import (
	"testing"
)

// FuzzParseTaskFile verifica que nenhum input causa panic no parser de tasks.md.
func FuzzParseTaskFile(f *testing.F) {
	// Seed corpus baseado em testdata/taskloop/valid/tasks.md
	f.Add([]byte(`# Resumo das Tarefas de Implementacao

## Tarefas

| # | Titulo | Status | Dependencias | Paralelizavel |
|---|--------|--------|-------------|---------------|
| 1.0 | Setup domain layer | pending | — | — |
| 2.0 | Implement ports | pending | 1.0 | Nao |
| 3.0 | Add adapters | pending | 1.0, 2.0 | Nao |
`))
	// Seed adversarial: testdata/taskloop/malformed/tasks.md
	f.Add([]byte("# Tasks malformado\n\nSem tabela valida aqui.\nApenas texto simples.\n"))
	// Seeds adicionais de borda
	f.Add([]byte(""))
	f.Add([]byte("| # | Titulo | Status | Dependencias | Paralelizavel |\n|---|--------|--------|-------------|---------------|\n"))
	f.Add([]byte("| 1.0 | Task A | pending | — | — |"))
	f.Add([]byte("| 1.0 | Task A | Concluido | — | — |\n| 2.0 | Task B | pendente | 1.0 | — |"))
	f.Add([]byte("| abc | titulo | status | deps | par |"))
	f.Add([]byte("| 1.0 |"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Erros sao aceitaveis; panic nao
		_, _ = ParseTasksFile(data)
	})
}

// FuzzReadTaskFileStatus verifica que nenhum input causa panic ao extrair status.
func FuzzReadTaskFileStatus(f *testing.F) {
	f.Add([]byte("# Task\n**Status:** pending\n**Prioridade:** Alta\n"))
	f.Add([]byte("# Task\n**Status:** Concluído (done)\n"))
	f.Add([]byte("**Status:** in_progress\n"))
	f.Add([]byte("# Apenas titulo\nSem campo de status.\n"))
	f.Add([]byte(""))
	f.Add([]byte("**Status:**\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = ReadTaskFileStatus(data)
	})
}
