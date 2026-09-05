// Package specdigest calcula o digest canonico de artefatos de especificacao.
//
// O digest identifica PRD, techspec e tasks aprovados. Ele precisa ser estavel
// entre plataformas: um checkout com core.autocrlf=true (padrao do Git for
// Windows) reescreve LF como CRLF e faria o mesmo artefato produzir digests
// diferentes conforme o sistema operacional, marcando artefatos aprovados como
// stale sem que ninguem os tenha editado.
package specdigest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

var (
	_crlf = []byte("\r\n")
	_lf   = []byte("\n")
)

// Canonical retorna o SHA-256 hexadecimal do conteudo com terminadores de linha
// normalizados para LF. Para conteudo ja em LF a normalizacao e um no-op, entao
// digests calculados antes desta normalizacao permanecem validos.
func Canonical(content []byte) string {
	sum := sha256.Sum256(Normalize(content))
	return hex.EncodeToString(sum[:])
}

// Normalize converte CRLF em LF. Um CR solitario e preservado: git nunca o
// produz por conversao de fim de linha, entao trata-lo seria alterar conteudo
// legitimo do artefato.
func Normalize(content []byte) []byte {
	if !bytes.Contains(content, _crlf) {
		return content
	}
	return bytes.ReplaceAll(content, _crlf, _lf)
}
