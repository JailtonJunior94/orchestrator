package aispecharness

import (
	"fmt"

	"github.com/spf13/cobra"
)

// requireFlag valida que uma flag obrigatoria esta presente e nao vazia.
// Diferencia flag ausente de flag presente mas vazia, retornando mensagem
// amigavel em PT-BR com exemplo de uso real do comando.
func requireFlag(cmd *cobra.Command, name, example string) error {
	f := cmd.Flags().Lookup(name)
	if f == nil || f.Value.String() == "" {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("flag --%s nao pode ficar vazia.\nExemplo:\n  %s", name, example)
		}
		return fmt.Errorf("flag --%s e obrigatoria.\nExemplo:\n  %s", name, example)
	}
	return nil
}
