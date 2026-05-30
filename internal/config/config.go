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

var _ SourceProvider = (*LocalSource)(nil)

func (s *LocalSource) SourceDir() string { return s.Dir }

// InstallScope define o escopo de instalacao: projeto (default) ou global.
// ADR-019.
type InstallScope string

const (
	// ScopeProject instala no diretorio do projeto (default).
	ScopeProject InstallScope = "project"
	// ScopeGlobal instala globalmente em dirs de config do usuario (~/.aispec/).
	ScopeGlobal InstallScope = "global"
)

// InstallOptions agrupa opcoes para o comando install.
type InstallOptions struct {
	ProjectDir             string
	SourceDir              string
	Tools                  []skills.Tool // OPCIONAL: vazio => auto-detect via AgentDetector (ADR-019)
	Langs                  []skills.Lang
	LinkMode               skills.LinkMode
	Scope                  InstallScope // novo (ADR-019): "project" (default) ou "global"
	DryRun                 bool
	GenerateCtx            bool
	CodexProfile           string
	FocusPaths             []string
	FollowExternalSymlinks bool
}

// UpgradeOptions agrupa opcoes para o comando upgrade.
type UpgradeOptions struct {
	ProjectDir             string
	SourceDir              string
	CheckOnly              bool
	Langs                  []skills.Lang
	CodexProfile           string
	FollowExternalSymlinks bool
}
