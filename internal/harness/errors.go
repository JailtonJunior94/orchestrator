package harness

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

type UnknownFieldError struct {
	Path   string
	Fields []string
}

func (e *UnknownFieldError) Error() string {
	location := e.Path
	if location == "" {
		location = "<root>"
	}
	quoted := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		quoted = append(quoted, fmt.Sprintf("%q", field))
	}
	return fmt.Sprintf(
		"harness contract has unknown field(s) %s at %q; supported fields are declared in harness-contract.schema.json; remove the field or fix the typo",
		strings.Join(quoted, ", "), location,
	)
}

type InvalidTypeError struct {
	Path     string
	Found    string
	Expected string
}

func (e *InvalidTypeError) Error() string {
	location := e.Path
	if location == "" {
		location = "<root>"
	}
	return fmt.Sprintf(
		"harness contract field %q has invalid type: found %s, expected %s; fix the value type in .agents/harness.yaml",
		location, e.Found, e.Expected,
	)
}

type UnsupportedVersionError struct {
	Found     any
	Supported int
}

func (e *UnsupportedVersionError) Error() string {
	found := "absent"
	if e.Found != nil {
		found = fmt.Sprintf("%v", e.Found)
	}
	return fmt.Sprintf(
		"harness contract version %s is not supported; supported version is %d; set \"version: %d\" in .agents/harness.yaml",
		found, e.Supported, e.Supported,
	)
}

type ErrorClassifier struct{}

func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

func (c *ErrorClassifier) Classify(err error, raw map[string]any) error {
	validationError, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return fmt.Errorf("harness contract is invalid: %s", strings.Join(strings.Fields(err.Error()), " "))
	}

	if typed := c.findVersionError(validationError, raw); typed != nil {
		return typed
	}
	if typed := c.findFirstCause(validationError, c.isAdditionalPropertiesCause); typed != nil {
		k := typed.ErrorKind.(*kind.AdditionalProperties)
		return &UnknownFieldError{
			Path:   strings.Join(typed.InstanceLocation, "."),
			Fields: k.Properties,
		}
	}
	if typed := c.findFirstCause(validationError, c.isTypeCause); typed != nil {
		k := typed.ErrorKind.(*kind.Type)
		return &InvalidTypeError{
			Path:     strings.Join(typed.InstanceLocation, "."),
			Found:    k.Got,
			Expected: strings.Join(k.Want, " or "),
		}
	}

	return fmt.Errorf("harness contract is invalid: %s", validationError.Error())
}

func (c *ErrorClassifier) findVersionError(root *jsonschema.ValidationError, raw map[string]any) *UnsupportedVersionError {
	if typed := c.findFirstCause(root, c.isVersionConstCause); typed != nil {
		k := typed.ErrorKind.(*kind.Const)
		return &UnsupportedVersionError{Found: c.normalizeScalar(k.Got), Supported: SupportedVersion}
	}
	if typed := c.findFirstCause(root, c.isVersionRequiredCause); typed != nil {
		return &UnsupportedVersionError{Found: c.normalizeScalar(raw["version"]), Supported: SupportedVersion}
	}
	return nil
}

func (c *ErrorClassifier) normalizeScalar(value any) any {
	number, ok := value.(json.Number)
	if !ok {
		return value
	}
	if integer, err := number.Int64(); err == nil {
		return integer
	}
	if float, err := number.Float64(); err == nil {
		return float
	}
	return number.String()
}

func (c *ErrorClassifier) isVersionConstCause(ve *jsonschema.ValidationError) bool {
	if len(ve.InstanceLocation) != 1 || ve.InstanceLocation[0] != "version" {
		return false
	}
	_, ok := ve.ErrorKind.(*kind.Const)
	return ok
}

func (c *ErrorClassifier) isVersionRequiredCause(ve *jsonschema.ValidationError) bool {
	required, ok := ve.ErrorKind.(*kind.Required)
	if !ok {
		return false
	}
	for _, missing := range required.Missing {
		if missing == "version" {
			return true
		}
	}
	return false
}

func (c *ErrorClassifier) isAdditionalPropertiesCause(ve *jsonschema.ValidationError) bool {
	_, ok := ve.ErrorKind.(*kind.AdditionalProperties)
	return ok
}

func (c *ErrorClassifier) isTypeCause(ve *jsonschema.ValidationError) bool {
	_, ok := ve.ErrorKind.(*kind.Type)
	return ok
}

func (c *ErrorClassifier) findFirstCause(root *jsonschema.ValidationError, match func(*jsonschema.ValidationError) bool) *jsonschema.ValidationError {
	if match(root) {
		return root
	}
	for _, cause := range root.Causes {
		if found := c.findFirstCause(cause, match); found != nil {
			return found
		}
	}
	return nil
}
