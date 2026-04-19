package aispecharness

import (
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Valida frontmatter de skills de governanca",
	Long: `Valida que todos os SKILL.md possuem frontmatter YAML valido
com campos obrigatorios: name, version, description.

Exemplos:
  ai-spec-harness validate .agents/skills
  ai-spec-harness validate`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillsDir := ".agents/skills"
		if len(args) > 0 {
			skillsDir = args[0]
		}

		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()

		if !fsys.IsDir(skillsDir) {
			return fmt.Errorf("diretorio de skills nao encontrado: %s", skillsDir)
		}

		entries, err := fsys.ReadDir(skillsDir)
		if err != nil {
			return err
		}

		errors := 0
		checked := 0

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillName := e.Name()
			skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")

			data, err := fsys.ReadFile(skillFile)
			if err != nil {
				continue
			}

			checked++
			fm := skills.ParseFrontmatter(data)

			if fm.Version == "" {
				printer.Error("%s — campo obrigatorio ausente: version", skillName)
				errors++
			} else if !isValidSemver(fm.Version) {
				printer.Error("%s — version nao segue semver: %s", skillName, fm.Version)
				errors++
			}

			if fm.Description == "" {
				printer.Error("%s — campo obrigatorio ausente: description", skillName)
				errors++
			}

			// Validar que name no frontmatter coincide com diretorio
			name := skills.ParseFrontmatterName(data)
			if name == "" {
				printer.Error("%s — campo obrigatorio ausente: name", skillName)
				errors++
			} else if name != skillName {
				printer.Error("%s — name no frontmatter (%s) difere do diretorio (%s)", skillName, name, skillName)
				errors++
			}

			for _, dep := range fm.DependsOn {
				if !fsys.IsDir(filepath.Join(skillsDir, dep)) {
					printer.Error("%s — depends_on '%s' nao encontrada", skillName, dep)
					errors++
				}
			}
		}

		if checked == 0 {
			return fmt.Errorf("nenhuma skill encontrada em %s", skillsDir)
		}

		if errors > 0 {
			printer.Info("")
			printer.Info("Validacao falhou: %d erro(s) em %d skill(s)", errors, checked)
			return fmt.Errorf("validacao falhou")
		}

		printer.Info("Validacao aprovada: %d skill(s) com frontmatter valido", checked)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func isValidSemver(v string) bool {
	parts := 0
	for _, c := range v {
		if c == '.' {
			parts++
		} else if c < '0' || c > '9' {
			if c == '-' {
				break // pre-release suffix ok
			}
			return false
		}
	}
	return parts >= 2
}
