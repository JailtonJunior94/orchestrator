package specs

import (
	"errors"
	"fmt"
	"strings"
)

// AccessMode descreve o nível de acesso solicitado ao agente Codex ACP.
// ADR-013 D-02.
type AccessMode string

const (
	// AccessModeRestricted é o modo padrão — acesso restrito ao diretório de trabalho.
	AccessModeRestricted AccessMode = "restricted"

	// AccessModeFull concede ao agente acesso completo ao sistema de arquivos.
	// Atenção: usar apenas em ambientes isolados (ex: sandbox/container). R-03.
	AccessModeFull AccessMode = "full"
)

type BootstrapArgsFunc func(model, reasoning string, addDirs []string, mode AccessMode, workDir string) []string

type WindowResolverFunc func(model string) ContextWindow

// FallbackLauncher descreve um launcher alternativo para iniciar o agente
// quando o binário canônico não estiver disponível no PATH.
type FallbackLauncher struct {
	Command   string
	FixedArgs []string
}

// Spec descreve uma configuração de runtime para um agente ACP.
// Construída apenas via construtores de catálogo (ex: Claude()) — não instanciar por literal (R-DDD-001).
type Spec struct {
	ID             string
	DisplayName    string
	Command        string
	FixedArgs      []string
	Fallbacks      []FallbackLauncher
	AccessModeFlag string
	// metadata para runtime_init e probe error (ADR-012 D-03)
	sdkVersion string
	npmVersion string
	npmPackage string
	// contextWindow é o VO de janela de contexto desta CLI (ADR-023).
	// Zero-value (MaxTokens==0) ⇒ WindowStandard ⇒ comportamento F1.
	contextWindow ContextWindow
	// bootstrapArgs gera argumentos dinâmicos de inicialização (ex: Codex).
	// nil significa no-op: BootstrapArgs() retorna nil. ADR-013 D-02/D-03.
	bootstrapArgs  BootstrapArgsFunc
	windowResolver WindowResolverFunc
}

// DriverID retorna o DriverID desta Spec (ADR-020).
// Retorna DriverID zero-value se o ID não for válido (uso ad-hoc; sem panic).
func (s Spec) DriverID() DriverID {
	d, _ := NewCatalog().ParseDriverID(s.ID)
	return d
}

// ContextWindow retorna o VO de janela de contexto desta Spec (ADR-023).
// Zero-value (MaxTokens==0) ⇒ WindowStandard ⇒ comportamento F1.
func (s Spec) ContextWindow() ContextWindow { return s.contextWindow }

func (s Spec) ResolveWindow(model string) ContextWindow {
	if s.windowResolver == nil {
		return s.contextWindow
	}
	return s.windowResolver(model)
}

// SDKVersion retorna a versão do SDK ACP Go associada a esta Spec.
func (s Spec) SDKVersion() string { return s.sdkVersion }

// NPMVersion retorna a versão npm pinada do agente ACP associada a esta Spec.
func (s Spec) NPMVersion() string { return s.npmVersion }

// NPMPackage retorna o nome do pacote npm do agente ACP associado a esta Spec.
func (s Spec) NPMPackage() string { return s.npmPackage }

// BootstrapArgs retorna os argumentos dinâmicos de inicialização do agente,
// delegando para bootstrapArgs se definido. Retorna nil quando bootstrapArgs == nil
// (comportamento no-op para Claude/Copilot). ADR-013 D-02.
func (s Spec) BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode, workDir string) []string {
	if s.bootstrapArgs == nil {
		return nil
	}
	return s.bootstrapArgs(model, reasoning, addDirs, mode, workDir)
}

var ErrFixedArgsFormat = errors.New("fixed args: positional item after flag item")

func ValidateFixedArgsFormat(args []string) error {
	sawFlag := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			sawFlag = true
			continue
		}
		if sawFlag {
			return fmt.Errorf("%w: %q after a flag item", ErrFixedArgsFormat, arg)
		}
	}
	return nil
}

// ValidateFallbackFixedArgsFormat aplica a invariante de formato de FixedArgs
// (ValidateFixedArgsFormat) ao segmento de args que o launcher de fallback repassa
// ao binário resolvido, ignorando o prefixo de flags/positional do próprio launcher
// (ex: "--yes" seguido do pacote npm no fallback via npx).
func ValidateFallbackFixedArgsFormat(args []string) error {
	pivot := -1
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			pivot = i
			break
		}
	}
	if pivot == -1 {
		return nil
	}
	if err := ValidateFixedArgsFormat(args[pivot+1:]); err != nil {
		return fmt.Errorf("fallback fixed args: %w", err)
	}
	return nil
}

func validateFallbacksFormat(id string, fallbacks []FallbackLauncher) {
	for _, fb := range fallbacks {
		if err := ValidateFallbackFixedArgsFormat(fb.FixedArgs); err != nil {
			panic(fmt.Sprintf("spec %q: fallback %q: %v", id, fb.Command, err))
		}
	}
}

// newSpec é o construtor interno, acessível apenas dentro do pacote.
// Consumidores externos devem usar funções de catálogo como Claude().
func (c *Catalog) newSpec(
	id, displayName, command string,
	fixedArgs []string,
	fallbacks []FallbackLauncher,
	accessModeFlag string,
	sdkVersion, npmVersion, npmPackage string,
	window ContextWindow,
) Spec {
	if err := ValidateFixedArgsFormat(fixedArgs); err != nil {
		panic(fmt.Sprintf("spec %q: %v", id, err))
	}
	validateFallbacksFormat(id, fallbacks)
	return Spec{
		ID:             id,
		DisplayName:    displayName,
		Command:        command,
		FixedArgs:      fixedArgs,
		Fallbacks:      fallbacks,
		AccessModeFlag: accessModeFlag,
		sdkVersion:     sdkVersion,
		npmVersion:     npmVersion,
		npmPackage:     npmPackage,
		contextWindow:  window,
	}
}

func (c *Catalog) newSpecWithBootstrap(
	id, displayName, command string,
	fixedArgs []string,
	fallbacks []FallbackLauncher,
	accessModeFlag string,
	sdkVersion, npmVersion, npmPackage string,
	bootstrapArgs BootstrapArgsFunc,
	window ContextWindow,
	windowResolver WindowResolverFunc,
) Spec {
	s := NewCatalog().newSpec(id, displayName, command, fixedArgs, fallbacks, accessModeFlag, sdkVersion, npmVersion, npmPackage, window)
	s.bootstrapArgs = bootstrapArgs
	s.windowResolver = windowResolver
	return s
}
