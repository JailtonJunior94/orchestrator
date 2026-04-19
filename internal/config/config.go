package config

import "github.com/JailtonJunior94/ai-spec-harness/internal/skills"

// SourceProvider resolve o caminho para o repositorio de governanca (fonte das skills).
type SourceProvider interface {
	SourceDir() string
}

// LocalSource aponta para um diretorio local como fonte.
type LocalSource struct {
	Dir string
}

func (s *LocalSource) SourceDir() string { return s.Dir }

// InstallOptions agrupa opcoes para o comando install.
type InstallOptions struct {
	ProjectDir   string
	SourceDir    string
	Tools        []skills.Tool
	Langs        []skills.Lang
	LinkMode     skills.LinkMode
	DryRun       bool
	GenerateCtx  bool
	CodexProfile string
	FocusPaths   []string
}

// UpgradeOptions agrupa opcoes para o comando upgrade.
type UpgradeOptions struct {
	ProjectDir   string
	SourceDir    string
	CheckOnly    bool
	Langs        []skills.Lang
	CodexProfile string
}
