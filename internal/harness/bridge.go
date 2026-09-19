package harness

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type YAMLBridge struct{}

func NewYAMLBridge() *YAMLBridge {
	return &YAMLBridge{}
}

func (b *YAMLBridge) Decode(data []byte) (map[string]any, error) {
	var decoded any
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("parse harness contract YAML: %w", err)
	}

	normalized, err := b.normalizeValue(decoded)
	if err != nil {
		return nil, err
	}

	root, ok := normalized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("harness contract root must be a mapping, got %T", normalized)
	}
	return root, nil
}

func (b *YAMLBridge) normalizeValue(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(v))
		for key, val := range v {
			normalizedVal, err := b.normalizeValue(val)
			if err != nil {
				return nil, err
			}
			normalized[key] = normalizedVal
		}
		return normalized, nil
	case map[any]any:
		normalized := make(map[string]any, len(v))
		for key, val := range v {
			stringKey, err := b.normalizeKey(key)
			if err != nil {
				return nil, err
			}
			normalizedVal, err := b.normalizeValue(val)
			if err != nil {
				return nil, err
			}
			normalized[stringKey] = normalizedVal
		}
		return normalized, nil
	case []any:
		normalized := make([]any, len(v))
		for i, item := range v {
			normalizedItem, err := b.normalizeValue(item)
			if err != nil {
				return nil, err
			}
			normalized[i] = normalizedItem
		}
		return normalized, nil
	default:
		return v, nil
	}
}

func (b *YAMLBridge) normalizeKey(key any) (string, error) {
	switch k := key.(type) {
	case string:
		return k, nil
	case fmt.Stringer:
		return k.String(), nil
	case bool, int, int64, uint64, float64:
		return fmt.Sprintf("%v", k), nil
	default:
		return "", fmt.Errorf("harness contract has non-normalizable key of type %T: %v", key, key)
	}
}
