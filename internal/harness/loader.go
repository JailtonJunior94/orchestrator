package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const ContractRelativePath = ".agents/harness.yaml"

type ContractDecoder struct {
	bridge    *YAMLBridge
	validator Validator
}

func NewContractDecoder(validator Validator) *ContractDecoder {
	return &ContractDecoder{bridge: NewYAMLBridge(), validator: validator}
}

func (d *ContractDecoder) Decode(data []byte) (Contract, error) {
	raw, err := d.bridge.Decode(data)
	if err != nil {
		return Contract{}, err
	}

	if err := d.validator.Validate(raw); err != nil {
		return Contract{}, err
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return Contract{}, fmt.Errorf("encode validated harness contract: %w", err)
	}

	var contract Contract
	if err := json.Unmarshal(encoded, &contract); err != nil {
		return Contract{}, fmt.Errorf("decode validated harness contract: %w", err)
	}
	return contract, nil
}

type Loader interface {
	Load(projectDir string) (Contract, Source, error)
}

type DefaultLoader struct {
	fs       fs.FileSystem
	decoder  *ContractDecoder
	fallback *EmbeddedDefault
}

var _ Loader = (*DefaultLoader)(nil)

func NewDefaultLoader(fileSystem fs.FileSystem) *DefaultLoader {
	return &DefaultLoader{
		fs:       fileSystem,
		decoder:  NewContractDecoder(NewSchemaValidator()),
		fallback: NewEmbeddedDefault(),
	}
}

func (l *DefaultLoader) Load(projectDir string) (Contract, Source, error) {
	path := filepath.Join(projectDir, filepath.FromSlash(ContractRelativePath))

	data, err := l.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			contract, defaultErr := l.fallback.Contract()
			if defaultErr != nil {
				return Contract{}, "", defaultErr
			}
			return contract, SourceDefault, nil
		}
		return Contract{}, "", fmt.Errorf("read harness contract %s: %w", path, err)
	}

	contract, err := l.decoder.Decode(data)
	if err != nil {
		return Contract{}, "", err
	}
	return contract, SourceFile, nil
}
