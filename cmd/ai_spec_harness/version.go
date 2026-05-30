package aispecharness

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	internalskills "github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
	"github.com/spf13/cobra"
)

type versionCommand struct{}

func newVersionCmd() *cobra.Command {
	handler := &versionCommand{}
	var skillsMode string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Exibe a versao do ai-spec-harness",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handler.run(cmd.OutOrStdout(), skillsMode, "")
		},
	}

	cmd.Flags().StringVar(&skillsMode, "skills", "", "Listar versoes de skills: embedded, installed, ou ambos (sem valor)")
	cmd.Flags().Lookup("skills").NoOptDefVal = "both"
	return cmd
}

// runVersion exibe a versao do CLI e, opcionalmente, as versoes das skills.
// installedDir sobrescreve o diretorio de busca de skills instaladas (usado em testes);
// se vazio, usa os.Getwd().
func (c *versionCommand) run(w io.Writer, mode string, installedDir string) error {
	fmt.Fprintf(w, "ai-spec-harness %s (commit: %s, built: %s)\n", version.NewProvider().Resolve("."), version.Commit, version.Date)

	if mode == "" {
		return nil
	}

	switch mode {
	case "embedded":
		skills, err := c.listEmbeddedSkills()
		if err != nil {
			return fmt.Errorf("listar skills embutidas: %w", err)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Embedded skills:")
		c.fprintSkills(w, skills)

	case "installed":
		dir, err := c.resolveInstalledDir(installedDir)
		if err != nil {
			return err
		}
		skills, err := c.listInstalledSkillsFromPath(dir)
		if err != nil {
			return fmt.Errorf("listar skills instaladas: %w", err)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Installed skills:")
		c.fprintSkills(w, skills)

	default: // "both"
		embSkills, err := c.listEmbeddedSkills()
		if err != nil {
			return fmt.Errorf("listar skills embutidas: %w", err)
		}
		dir, err := c.resolveInstalledDir(installedDir)
		if err != nil {
			return err
		}
		instSkills, err := c.listInstalledSkillsFromPath(dir)
		if err != nil {
			return fmt.Errorf("listar skills instaladas: %w", err)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Embedded skills:")
		c.fprintSkills(w, embSkills)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Installed skills:")
		c.fprintSkills(w, instSkills)
	}

	return nil
}

func (c *versionCommand) resolveInstalledDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("obter diretorio atual: %w", err)
	}
	return cwd, nil
}

type skillEntry struct {
	Name    string
	Version string
}

func (c *versionCommand) listEmbeddedSkills() ([]skillEntry, error) {
	const root = "assets/.agents/skills"

	entries, err := embedded.Assets.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var skills []skillEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		skillMdPath := root + "/" + skillName + "/SKILL.md"
		data, err := embedded.Assets.ReadFile(skillMdPath)
		if err != nil {
			skills = append(skills, skillEntry{Name: skillName})
			continue
		}
		fm := internalskills.NewCatalog().ParseFrontmatter(data)
		name := fm.Name
		if name == "" {
			name = skillName
		}
		skills = append(skills, skillEntry{Name: name, Version: fm.Version})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func (c *versionCommand) listInstalledSkillsFromPath(dir string) ([]skillEntry, error) {
	skillsDir := filepath.Join(dir, ".agents", "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	var skills []skillEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		skillMdPath := filepath.Join(skillsDir, skillName, "SKILL.md")
		data, err := os.ReadFile(skillMdPath)
		if err != nil {
			skills = append(skills, skillEntry{Name: skillName})
			continue
		}
		fm := internalskills.NewCatalog().ParseFrontmatter(data)
		name := fm.Name
		if name == "" {
			name = skillName
		}
		skills = append(skills, skillEntry{Name: name, Version: fm.Version})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func (c *versionCommand) fprintSkills(w io.Writer, skills []skillEntry) {
	if len(skills) == 0 {
		fmt.Fprintln(w, "  nenhuma skill instalada")
		return
	}
	for _, s := range skills {
		ver := s.Version
		if ver == "" {
			ver = "(sem versao)"
		}
		fmt.Fprintf(w, "  %-30s %s\n", s.Name, ver)
	}
}
