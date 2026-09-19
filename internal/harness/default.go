package harness

import (
	_ "embed"
	"fmt"
)

//go:embed default-contract.yaml
var defaultContractYAML []byte

type EmbeddedDefault struct {
	decoder *ContractDecoder
}

func NewEmbeddedDefault() *EmbeddedDefault {
	return &EmbeddedDefault{decoder: NewContractDecoder(NewSchemaValidator())}
}

func (d *EmbeddedDefault) Contract() (Contract, error) {
	contract, err := d.decoder.Decode(defaultContractYAML)
	if err != nil {
		return Contract{}, fmt.Errorf("embedded default harness contract is invalid: %w", err)
	}
	return contract, nil
}
