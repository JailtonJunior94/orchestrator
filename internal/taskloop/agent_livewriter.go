package taskloop

import (
	"io"
	"sync"
)

// syncWriter serializa Write em torno de um io.Writer subjacente para uso
// seguro por multiplas goroutines (ex: stdout + stderr de um exec.Cmd
// compartilhando o mesmo destino live, onde o writer pode nao ser
// thread-safe — bytes.Buffer e o caso mais comum em testes).
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
