package harness

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
)

type DuplicateKeyChecker struct {
	inspector *SchemaInspector
}

func NewDuplicateKeyChecker() *DuplicateKeyChecker {
	return &DuplicateKeyChecker{inspector: NewSchemaInspector()}
}

func (c *DuplicateKeyChecker) OperationalKeyNames() []string {
	runtimeType := reflect.TypeOf(config.Runtime{})
	names := make([]string, 0, runtimeType.NumField())

	for i := range runtimeType.NumField() {
		field := runtimeType.Field(i)
		tag, ok := field.Tag.Lookup("yaml")
		if !ok || tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func (c *DuplicateKeyChecker) findDuplicateKeys(schemaKeys, operationalKeys []string) []string {
	operationalSet := make(map[string]struct{}, len(operationalKeys))
	for _, name := range operationalKeys {
		operationalSet[name] = struct{}{}
	}

	var duplicates []string
	for _, name := range schemaKeys {
		if _, exists := operationalSet[name]; exists {
			duplicates = append(duplicates, name)
		}
	}
	slices.Sort(duplicates)
	return duplicates
}

func (c *DuplicateKeyChecker) Check() error {
	schemaKeys, err := c.inspector.KeyNames()
	if err != nil {
		return fmt.Errorf("check harness contract key duplication: %w", err)
	}

	duplicates := c.findDuplicateKeys(schemaKeys, c.OperationalKeyNames())
	if len(duplicates) > 0 {
		return fmt.Errorf(
			"harness contract schema declares key(s) %s that already exist as operational key(s) in internal/config.Runtime; "+
				"policy keys must never duplicate operational keys (RF-05); rename the harness contract field",
			strings.Join(duplicates, ", "),
		)
	}
	return nil
}
