package inspect

import (
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// Service inspeciona o estado de instalacao de governanca.
type Service struct {
	fs       fs.FileSystem
	printer  *output.Printer
	manifest *manifest.Store
	detector detect.Detector
}

func NewService(fsys fs.FileSystem, printer *output.Printer, mfst *manifest.Store, det detect.Detector) *Service {
	return &Service{fs: fsys, printer: printer, manifest: mfst, detector: det}
}

func (s *Service) Execute(projectDir string) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if !s.fs.IsDir(absDir) {
		return fmt.Errorf("diretorio nao encontrado: %s", absDir)
	}

	s.printer.Info("Inspecionando: %s", absDir)
	s.printer.Info("")

	// Verificar manifesto
	if s.manifest.Exists(absDir) {
		mf, err := s.manifest.Load(absDir)
		if err != nil {
			s.printer.Warn("Manifesto corrompido: %v", err)
		} else {
			s.printer.Info("Manifesto (.ai_spec_harness.json):")
			s.printer.Info("  Versao ai-spec-harness:  %s", mf.Version)
			s.printer.Info("  Instalado em:  %s", mf.CreatedAt.Format("2006-01-02 15:04:05"))
			s.printer.Info("  Atualizado em: %s", mf.UpdatedAt.Format("2006-01-02 15:04:05"))
			s.printer.Info("  Fonte:         %s", mf.SourceDir)
			s.printer.Info("  Modo:          %s", mf.LinkMode)
			s.printer.Info("  Ferramentas:   %v", toolNames(mf.Tools))
			s.printer.Info("  Linguagens:    %v", langNames(mf.Langs))
			s.printer.Info("  Skills:        %d", len(mf.Skills))
			s.printer.Info("")
		}
	} else {
		s.printer.Info("Manifesto: nao encontrado (instalacao pode ter sido feita via shell)")
	}

	// Skills instaladas
	skillsDir := filepath.Join(absDir, ".agents", "skills")
	if s.fs.IsDir(skillsDir) {
		entries, err := s.fs.ReadDir(skillsDir)
		if err == nil {
			s.printer.Info("Skills instaladas:")
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				skillPath := filepath.Join(skillsDir, e.Name())
				mode := "copy"
				if s.fs.IsSymlink(skillPath) {
					mode = "symlink"
				}

				ver := ""
				skillMD := filepath.Join(skillPath, "SKILL.md")
				if data, err := s.fs.ReadFile(skillMD); err == nil {
					fm := skills.ParseFrontmatter(data)
					if fm.Version != "" {
						ver = fm.Version
					}
				}
				s.printer.Info("  %-35s %s [%s]", e.Name(), ver, mode)
			}
			s.printer.Info("")
		}
	} else {
		s.printer.Info("Skills: nenhuma instalada (.agents/skills/ ausente)")
	}

	// Ferramentas detectadas
	tools := s.detector.DetectTools(absDir)
	s.printer.Info("Ferramentas detectadas: %v", toolNames(tools))

	// Linguagens detectadas
	langs := s.detector.DetectLangs(absDir)
	s.printer.Info("Linguagens detectadas: %v", langNames(langs))

	return nil
}

func toolNames(tools []skills.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = string(t)
	}
	return out
}

func langNames(langs []skills.Lang) []string {
	out := make([]string, len(langs))
	for i, l := range langs {
		out[i] = string(l)
	}
	return out
}
