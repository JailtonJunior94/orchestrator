package specs

// launcherKind representa o tipo de launcher (binário ou npx).
type launcherKind string

const (
	launcherBinary launcherKind = "binary"
	launcherNpx    launcherKind = "npx"
)

// Launcher é um value object imutável que representa como um agente ACP é iniciado.
// Use NewBinaryLauncher ou NewNpxLauncher para construir.
type Launcher struct {
	kind launcherKind
	cmd  string
	args []string
}

// NewBinaryLauncher cria um Launcher que aponta para um binário no PATH.
func NewBinaryLauncher(cmd string, args ...string) Launcher {
	return Launcher{
		kind: launcherBinary,
		cmd:  cmd,
		args: args,
	}
}

// NewNpxLauncher cria um Launcher que invoca um pacote npm via npx.
// pkg é o nome do pacote (ex: "@agentclientprotocol/claude-agent-acp") e
// version é a versão pinada (ex: "0.1.2").
func NewNpxLauncher(pkg, version string) Launcher {
	return Launcher{
		kind: launcherNpx,
		cmd:  "npx",
		args: []string{"--yes", pkg + "@" + version},
	}
}

// Kind retorna "binary" ou "npx".
func (l Launcher) Kind() string {
	return string(l.kind)
}

// Command retorna o comando principal e os argumentos fixos do launcher.
func (l Launcher) Command() (string, []string) {
	return l.cmd, l.args
}
