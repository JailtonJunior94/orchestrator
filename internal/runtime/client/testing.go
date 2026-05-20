package client

import (
	"io"
)

// pipeIOProvider implementa IOProvider usando io.ReadWriter pré-construído.
// Usado exclusivamente em testes para injetar um PipeConn do acpfake.
type pipeIOProvider struct {
	w io.Writer
	r io.Reader
}

func (p *pipeIOProvider) Provide() (io.Writer, io.Reader, error) {
	return p.w, p.r, nil
}

// NewTestClient cria um Client que usa os pipes w/r fornecidos em vez de spawn subprocess.
// Exclusivo para testes in-process com acpfake.
func NewTestClient(workDir string, w io.Writer, r io.Reader) Client {
	return newACPClient(workDir, &pipeIOProvider{w: w, r: r})
}
