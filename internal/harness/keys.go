package harness

import "encoding/json"

type schemaDoc struct {
	Properties map[string]json.RawMessage `json:"properties"`
}

type SchemaInspector struct{}

func NewSchemaInspector() *SchemaInspector {
	return &SchemaInspector{}
}

func (i *SchemaInspector) KeyNames() ([]string, error) {
	names := map[string]struct{}{}
	if err := i.collectKeyNames(contractSchemaJSON, names); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result, nil
}

func (i *SchemaInspector) collectKeyNames(rawSchema []byte, out map[string]struct{}) error {
	var doc schemaDoc
	if err := json.Unmarshal(rawSchema, &doc); err != nil {
		return err
	}
	for name, rawProperty := range doc.Properties {
		out[name] = struct{}{}
		if err := i.collectKeyNames(rawProperty, out); err != nil {
			return err
		}
	}
	return nil
}
