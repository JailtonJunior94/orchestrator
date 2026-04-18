package platform

import "runtime"

// Info contem informacoes sobre a plataforma de execucao.
type Info struct {
	OS   string
	Arch string
}

func Current() Info {
	return Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// SupportsSymlinks retorna true se a plataforma suporta symlinks nativamente.
// No Windows, symlinks exigem permissoes elevadas ou Developer Mode.
func (i Info) SupportsSymlinks() bool {
	return i.OS != "windows"
}
