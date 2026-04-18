package doctor

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// Check representa um diagnostico individual.
type Check struct {
	Name   string
	Status string // "ok", "warn", "fail"
	Detail string
}

// Service executa diagnosticos de saude da instalacao.
type Service struct {
	fs       fs.FileSystem
	printer  *output.Printer
	manifest *manifest.Store
	git      git.Repository
}

func NewService(fsys fs.FileSystem, printer *output.Printer, mfst *manifest.Store, gitRepo git.Repository) *Service {
	return &Service{fs: fsys, printer: printer, manifest: mfst, git: gitRepo}
}

func (s *Service) Execute(projectDir string) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if !s.fs.IsDir(absDir) {
		return fmt.Errorf("diretorio nao encontrado: %s", absDir)
	}

	s.printer.Info("Diagnosticando: %s", absDir)
	s.printer.Info("")

	checks := s.runChecks(absDir)

	var failCount int
	for _, c := range checks {
		icon := "OK"
		switch c.Status {
		case "warn":
			icon = "AVISO"
		case "fail":
			icon = "FALHA"
			failCount++
		}
		s.printer.Info("  [%-5s] %-35s %s", icon, c.Name, c.Detail)
	}

	s.printer.Info("")
	if failCount > 0 {
		s.printer.Info("Resultado: %d problema(s) encontrado(s)", failCount)
		return fmt.Errorf("%d problema(s) detectado(s)", failCount)
	}
	s.printer.Info("Resultado: tudo ok")
	return nil
}

func (s *Service) runChecks(projectDir string) []Check {
	var checks []Check

	// 1. Repositorio git valido
	checks = append(checks, s.checkGit(projectDir))

	// 2. Diretorio de skills existe
	checks = append(checks, s.checkSkillsDir(projectDir))

	// 3. Manifesto presente
	checks = append(checks, s.checkManifest(projectDir))

	// 4. Links simbolicos validos
	checks = append(checks, s.checkSymlinks(projectDir)...)

	// 5. Permissoes de escrita
	checks = append(checks, s.checkPermissions(projectDir))

	// 6. Git instalado
	checks = append(checks, s.checkGitBinary())

	return checks
}

func (s *Service) checkGit(projectDir string) Check {
	if s.git.IsRepo(projectDir) {
		return Check{Name: "Repositorio git", Status: "ok", Detail: "valido"}
	}
	return Check{Name: "Repositorio git", Status: "warn", Detail: "nao e um repositorio git"}
}

func (s *Service) checkSkillsDir(projectDir string) Check {
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if s.fs.IsDir(skillsDir) {
		entries, err := s.fs.ReadDir(skillsDir)
		if err == nil {
			count := 0
			for _, e := range entries {
				if e.IsDir() {
					count++
				}
			}
			return Check{Name: "Diretorio de skills", Status: "ok", Detail: fmt.Sprintf("%d skills encontradas", count)}
		}
	}
	return Check{Name: "Diretorio de skills", Status: "fail", Detail: ".agents/skills/ ausente"}
}

func (s *Service) checkManifest(projectDir string) Check {
	if s.manifest.Exists(projectDir) {
		mf, err := s.manifest.Load(projectDir)
		if err != nil {
			return Check{Name: "Manifesto", Status: "warn", Detail: fmt.Sprintf("corrompido: %v", err)}
		}
		return Check{Name: "Manifesto", Status: "ok", Detail: fmt.Sprintf("versao %s", mf.Version)}
	}
	return Check{Name: "Manifesto", Status: "warn", Detail: ".ai_spec_harness.json nao encontrado"}
}

func (s *Service) checkSymlinks(projectDir string) []Check {
	var checks []Check
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if !s.fs.IsDir(skillsDir) {
		return checks
	}

	entries, err := s.fs.ReadDir(skillsDir)
	if err != nil {
		return checks
	}

	brokenCount := 0
	for _, e := range entries {
		path := filepath.Join(skillsDir, e.Name())
		if s.fs.IsSymlink(path) {
			if !s.fs.Exists(path) {
				brokenCount++
			}
		}
	}

	if brokenCount > 0 {
		checks = append(checks, Check{
			Name:   "Symlinks de skills",
			Status: "fail",
			Detail: fmt.Sprintf("%d symlink(s) quebrado(s)", brokenCount),
		})
	} else {
		checks = append(checks, Check{
			Name:   "Symlinks de skills",
			Status: "ok",
			Detail: "todos os links validos",
		})
	}

	return checks
}

func (s *Service) checkPermissions(projectDir string) Check {
	if s.fs.Writable(projectDir) {
		return Check{Name: "Permissoes de escrita", Status: "ok", Detail: "diretorio gravavel"}
	}
	return Check{Name: "Permissoes de escrita", Status: "fail", Detail: "sem permissao de escrita"}
}

func (s *Service) checkGitBinary() Check {
	if _, err := exec.LookPath("git"); err != nil {
		return Check{Name: "Git instalado", Status: "fail", Detail: "git nao encontrado no PATH"}
	}
	return Check{Name: "Git instalado", Status: "ok", Detail: "disponivel"}
}
