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
}

// newSpec é o construtor interno, acessível apenas dentro do pacote.
// Consumidores externos devem usar funções de catálogo como Claude().
func newSpec(
	id, displayName, command string,
	fixedArgs []string,
	fallbacks []FallbackLauncher,
	accessModeFlag string,
) Spec {
	return Spec{
		ID:             id,
		DisplayName:    displayName,
		Command:        command,
		FixedArgs:      fixedArgs,
		Fallbacks:      fallbacks,
		AccessModeFlag: accessModeFlag,
	}
}
