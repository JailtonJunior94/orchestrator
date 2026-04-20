package skills

import (
	"testing"
)

// FuzzParseFrontmatter verifica que nenhum input causa panic no parser de frontmatter.
func FuzzParseFrontmatter(f *testing.F) {
	// Seed corpus baseado em SKILL.md reais do repositorio
	f.Add([]byte("---\nname: analyze-project\nversion: 1.2.3\ndescription: Analisa e classifica projetos.\n---\n\n# Skill\n"))
	f.Add([]byte("---\nname: execute-task\nversion: 1.0.0\ndescription: Executa tarefas.\ndepends_on: [review, bugfix]\nlang: go\nlink_mode: inline\nmax_depth: 2\n---\n"))
	f.Add([]byte("---\nname: go-implementation\nversion: 2.0.0\ndescription: Implementa codigo Go.\ntriggers: [feat, fix]\n---\n"))
	f.Add([]byte("---\nname: review\nversion: 1.5.0\ndescription: Revisa codigo.\ndepends_on: []\n---\n"))
	f.Add([]byte("# Sem frontmatter\nConteudo sem delimitadores."))
	f.Add([]byte("---\n---\n"))
	f.Add([]byte("---\nversion: 1.0.0\n---\n"))
	f.Add([]byte(""))
	f.Add([]byte("---"))
	f.Add([]byte("---\nmax_depth: nao-numero\n---\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Nao deve causar panic
		_ = ParseFrontmatter(data)
	})
}

// FuzzValidateFrontmatter verifica que nenhum input causa panic no validador de frontmatter.
func FuzzValidateFrontmatter(f *testing.F) {
	f.Add([]byte("---\nname: my-skill\nversion: 1.0.0\ndescription: Uma skill valida.\n---\n"), "my-skill")
	f.Add([]byte("---\nname: analyze-project\nversion: 1.2.3\ndescription: Analisa projetos.\ndepends_on: [review]\n---\n"), "analyze-project")
	f.Add([]byte("# sem frontmatter"), "")
	f.Add([]byte("---\n---\n"), "")
	f.Add([]byte("---\nversion: not-semver\ndescription: Desc.\n---\n"), "")
	f.Add([]byte(""), "")

	f.Fuzz(func(t *testing.T, data []byte, dirName string) {
		// Erros sao aceitaveis; panic nao
		_ = ValidateFrontmatter(data, dirName, nil)
	})
}
