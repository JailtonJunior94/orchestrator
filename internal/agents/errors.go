package agents

import "errors"

// ErrFrontmatterInvalid e retornado quando o frontmatter de um AGENT.md falha na validacao JSON Schema.
var ErrFrontmatterInvalid = errors.New("frontmatter invalido")

// ErrNameDirMismatch e retornado quando o campo name do frontmatter nao coincide com o nome do diretorio pai (RF-06).
var ErrNameDirMismatch = errors.New("name no frontmatter diverge do nome do diretorio")

// ErrVersionInvalid e retornado quando o campo version nao e um SemVer valido.
var ErrVersionInvalid = errors.New("version invalida: deve seguir SemVer (X.Y.Z)")

// ErrIDEUnsupported e retornado quando o campo runtime.ide contem um valor fora do enum permitido.
var ErrIDEUnsupported = errors.New("runtime.ide invalido: valores aceitos sao claude, codex, gemini, copilot")

// ErrAgentNotFound e retornado quando um agente referenciado nao e encontrado em nenhum escopo.
var ErrAgentNotFound = errors.New("agente nao encontrado")
