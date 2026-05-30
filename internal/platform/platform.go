package platform

import "runtime"

// Info contem informacoes sobre a plataforma de execucao.
type Info struct {
	OS   string
	Arch string
}

// Detector detecta informações da plataforma de execução.
type Detector struct{}

// NewDetector cria um Detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Current retorna as informações da plataforma de execução atual.
func (d *Detector) Current() Info {
	return Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// SupportsSymlinks retorna true se a plataforma suporta symlinks nativamente.
// No Windows, symlinks exigem permissoes elevadas ou Developer Mode.
func (i Info) SupportsSymlinks() bool {
	return i.OS != "windows"
}
