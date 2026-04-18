package output

import (
	"fmt"
	"io"
	"os"
)

// Printer gerencia saida para o usuario com suporte a verbose.
type Printer struct {
	Out     io.Writer
	Err     io.Writer
	Verbose bool
}

func New(verbose bool) *Printer {
	return &Printer{Out: os.Stdout, Err: os.Stderr, Verbose: verbose}
}

func (p *Printer) Info(format string, args ...any) {
	fmt.Fprintf(p.Out, format+"\n", args...)
}

func (p *Printer) Step(format string, args ...any) {
	fmt.Fprintf(p.Out, "-> "+format+"\n", args...)
}

func (p *Printer) Debug(format string, args ...any) {
	if p.Verbose {
		fmt.Fprintf(p.Out, "   [debug] "+format+"\n", args...)
	}
}

func (p *Printer) Warn(format string, args ...any) {
	fmt.Fprintf(p.Err, "AVISO: "+format+"\n", args...)
}

func (p *Printer) Error(format string, args ...any) {
	fmt.Fprintf(p.Err, "ERRO: "+format+"\n", args...)
}

func (p *Printer) DryRun(format string, args ...any) {
	fmt.Fprintf(p.Out, "  [dry-run] "+format+"\n", args...)
}

// Status imprime uma linha de status estilo upgrade (OK, DESATUALIZADA, etc).
func (p *Printer) Status(status, skill, detail string) {
	fmt.Fprintf(p.Out, "  %-20s %s (%s)\n", status, skill, detail)
}
