package events

// AgentMessagePayload contém os dados de uma mensagem do agente.
type AgentMessagePayload struct {
	text string
}

// Text retorna o texto da mensagem.
func (p *AgentMessagePayload) Text() string {
	return p.text
}

// AgentThoughtPayload contém os dados de um pensamento do agente.
type AgentThoughtPayload struct {
	text string
}

// Text retorna o texto do pensamento.
func (p *AgentThoughtPayload) Text() string {
	return p.text
}

// ToolCallStartPayload contém os dados do início de uma chamada de ferramenta.
type ToolCallStartPayload struct {
	name  string
	input string
}

// Name retorna o nome da ferramenta chamada.
func (p *ToolCallStartPayload) Name() string {
	return p.name
}

// Input retorna os dados de entrada da chamada.
func (p *ToolCallStartPayload) Input() string {
	return p.input
}

// ToolCallUpdatePayload contém os dados de uma atualização de chamada de ferramenta.
type ToolCallUpdatePayload struct {
	output string
	status string
	final  bool
}

// Output retorna a saída parcial ou final da chamada.
func (p *ToolCallUpdatePayload) Output() string {
	return p.output
}

// Status retorna o status atual da chamada.
func (p *ToolCallUpdatePayload) Status() string {
	return p.status
}

// Final retorna true quando esta é a última atualização da chamada.
func (p *ToolCallUpdatePayload) Final() bool {
	return p.final
}

// SessionEndPayload contém os dados do fim de sessão.
type SessionEndPayload struct {
	exitCode int
	reason   string
}

// ExitCode retorna o código de saída da sessão.
func (p *SessionEndPayload) ExitCode() int {
	return p.exitCode
}

// Reason retorna o motivo de encerramento.
func (p *SessionEndPayload) Reason() string {
	return p.reason
}

// RuntimeInitPayload contém os dados de inicialização do runtime.
type RuntimeInitPayload struct {
	launcher   string
	command    string
	args       []string
	sdkVersion string
	npmVersion string
}

// Launcher retorna o tipo de launcher utilizado ("binary" ou "npx").
func (p *RuntimeInitPayload) Launcher() string {
	return p.launcher
}

// Command retorna o comando executado.
func (p *RuntimeInitPayload) Command() string {
	return p.command
}

// Args retorna os argumentos do comando.
func (p *RuntimeInitPayload) Args() []string {
	cp := make([]string, len(p.args))
	copy(cp, p.args)
	return cp
}

// SDKVersion retorna a versão do SDK ACP.
func (p *RuntimeInitPayload) SDKVersion() string {
	return p.sdkVersion
}

// NpmVersion retorna a versão do pacote npm (vazia quando launcher=binary).
func (p *RuntimeInitPayload) NpmVersion() string {
	return p.npmVersion
}

// UnknownPayload contém os dados de um evento de kind desconhecido.
type UnknownPayload struct {
	rawKind string
}

// RawKind retorna o valor bruto do kind desconhecido.
func (p *UnknownPayload) RawKind() string {
	return p.rawKind
}
