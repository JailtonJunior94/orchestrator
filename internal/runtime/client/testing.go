package client

import (
	"io"
	"time"
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
// Usa defaults: cap=64, publishTimeout=0 (F1 default, byte-equivalente ao comportamento atual).
func NewTestClient(workDir string, w io.Writer, r io.Reader) Client {
	return newACPClient(workDir, &pipeIOProvider{w: w, r: r}, defaultChannelCap, 0)
}

// NewTestClientWithBackpressure cria um Client de teste com capacidade e publishTimeout configuráveis.
// Permite testar cenários de backpressure (drop e slow-publish) em testes unitários.
func NewTestClientWithBackpressure(workDir string, w io.Writer, r io.Reader, cap int, publishTimeout time.Duration) Client {
	return newACPClient(workDir, &pipeIOProvider{w: w, r: r}, cap, publishTimeout)
}
