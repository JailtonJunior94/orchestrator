package specs

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

// BootstrapArgsFunc é o tipo da função que gera os argumentos de inicialização
// dinâmicos para um agente ACP. Codex requer argumentos que dependem de
// model, reasoning, addDirs e mode em tempo de execução.
// ADR-013 D-03.
type BootstrapArgsFunc func(model, reasoning string, addDirs []string, mode AccessMode) []string

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
	// bootstrapArgs gera argumentos dinâmicos de inicialização (ex: Codex).
	// nil significa no-op: BootstrapArgs() retorna nil. ADR-013 D-02/D-03.
	bootstrapArgs BootstrapArgsFunc
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
func (s Spec) BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string {
	if s.bootstrapArgs == nil {
		return nil
	}
	return s.bootstrapArgs(model, reasoning, addDirs, mode)
}

// newSpec é o construtor interno, acessível apenas dentro do pacote.
// Consumidores externos devem usar funções de catálogo como Claude().
func newSpec(
	id, displayName, command string,
	fixedArgs []string,
	fallbacks []FallbackLauncher,
	accessModeFlag string,
	sdkVersion, npmVersion, npmPackage string,
) Spec {
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
	}
}

// newSpecWithBootstrap é o variant constructor para specs que injetam uma
// BootstrapArgsFunc dinâmica (ex: Codex). ADR-013 D-03.
func newSpecWithBootstrap(
	id, displayName, command string,
	fixedArgs []string,
	fallbacks []FallbackLauncher,
	accessModeFlag string,
	sdkVersion, npmVersion, npmPackage string,
	bootstrapArgs BootstrapArgsFunc,
) Spec {
	s := newSpec(id, displayName, command, fixedArgs, fallbacks, accessModeFlag, sdkVersion, npmVersion, npmPackage)
	s.bootstrapArgs = bootstrapArgs
	return s
}
