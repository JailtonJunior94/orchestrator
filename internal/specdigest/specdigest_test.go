package specdigest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCanonical_IgualParaLFeCRLF(t *testing.T) {
	lf := []byte("# PRD\n\n- RF-01: exemplo\n")
	crlf := []byte("# PRD\r\n\r\n- RF-01: exemplo\r\n")

	if Canonical(lf) != Canonical(crlf) {
		t.Fatalf("digest divergiu entre LF e CRLF: %s != %s", Canonical(lf), Canonical(crlf))
	}
}

func TestCanonical_NaoAlteraDigestDeConteudoLF(t *testing.T) {
	// Garantia de nao regressao: para conteudo sem CRLF o digest canonico
	// precisa ser identico ao SHA-256 puro, senao todo hash ja aprovado
	// passaria a divergir.
	content := []byte("# PRD\n\n- RF-01: exemplo\n")
	sum := sha256.Sum256(content)

	if got, want := Canonical(content), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("Canonical alterou o digest de conteudo LF: %s != %s", got, want)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{name: "sem crlf permanece intacto", in: []byte("a\nb\n"), want: []byte("a\nb\n")},
		{name: "crlf vira lf", in: []byte("a\r\nb\r\n"), want: []byte("a\nb\n")},
		{name: "misto normaliza apenas crlf", in: []byte("a\r\nb\n"), want: []byte("a\nb\n")},
		{name: "cr solitario e preservado", in: []byte("a\rb"), want: []byte("a\rb")},
		{name: "vazio", in: []byte(""), want: []byte("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); !bytes.Equal(got, tt.want) {
				t.Fatalf("Normalize(%q) = %q, quer %q", tt.in, got, tt.want)
			}
		})
	}
}
