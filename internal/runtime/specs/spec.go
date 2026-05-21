package specs

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
}

// SDKVersion retorna a versão do SDK ACP Go associada a esta Spec.
func (s Spec) SDKVersion() string { return s.sdkVersion }

// NPMVersion retorna a versão npm pinada do agente ACP associada a esta Spec.
func (s Spec) NPMVersion() string { return s.npmVersion }

// NPMPackage retorna o nome do pacote npm do agente ACP associado a esta Spec.
func (s Spec) NPMPackage() string { return s.npmPackage }

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
