package output

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Processor orchestrates extraction, fixing, and validation of provider outputs.
type Processor interface {
	Process(ctx context.Context, raw string, schema []byte, options ProcessOptions) (*Result, error)
}

// Result contains the normalized provider output.
type Result struct {
	Markdown         string
	JSON             json.RawMessage
	Corrected        bool
	ValidationStatus string
	ValidationReport json.RawMessage
}

// ProcessOptions configures how provider output should be interpreted.
type ProcessOptions struct {
	RequireStructured bool
	SchemaName        string
	// OutputFormat signals the encoding of the raw provider output.
	// Set to "jsonl" when the provider emits JSONL (e.g. Codex with --json).
	OutputFormat string
}

// ProcessError surfaces whether the failure is recoverable via retry.
type ProcessError struct {
	Err         error
	Recoverable bool
}

func (e *ProcessError) Error() string {
	return e.Err.Error()
}

func (e *ProcessError) Unwrap() error {
	return e.Err
}

// IsRecoverable reports whether a processing error should trigger a provider retry.
func IsRecoverable(err error) bool {
	var processErr *ProcessError
	return errors.As(err, &processErr) && processErr.Recoverable
}

type outputProcessor struct{}

// NewProcessor creates the default output processor.
func NewProcessor() Processor {
	return outputProcessor{}
}

// Process normalizes the provider output and extracts the structured JSON payload.
func (outputProcessor) Process(ctx context.Context, raw string, schema []byte, options ProcessOptions) (*Result, error) {
	markdown := strings.TrimSpace(raw)

	if options.OutputFormat == "json" {
		return processProviderJSON(ctx, raw, schema, options)
	}

	if options.OutputFormat == "jsonl" {
		return processJSONL(ctx, raw, schema, options)
	}

	jsonText, err := ExtractJSON(raw)
	if err != nil {
		if !options.RequireStructured {
			report := buildValidationReport("not_applicable", false, options.SchemaName, false)
			return &Result{
				Markdown:         markdown,
				ValidationStatus: "not_applicable",
				ValidationReport: report,
			}, nil
		}

		return nil, &ProcessError{Err: err, Recoverable: true}
	}

	corrected := false
	if err := ValidateJSON(ctx, []byte(jsonText), schema); err != nil {
		if !errors.Is(err, ErrInvalidJSON) {
			return nil, &ProcessError{Err: err, Recoverable: false}
		}

		fixed := FixJSON(jsonText)
		if fixed == jsonText {
			return nil, &ProcessError{Err: err, Recoverable: true}
		}

		if validateErr := ValidateJSON(ctx, []byte(fixed), schema); validateErr != nil {
			recoverable := errors.Is(validateErr, ErrInvalidJSON)
			return nil, &ProcessError{Err: validateErr, Recoverable: recoverable}
		}

		jsonText = fixed
		corrected = true
	}

	status := "passed"
	if corrected {
		status = "corrected"
	}

	return &Result{
		Markdown:         markdown,
		JSON:             json.RawMessage(jsonText),
		Corrected:        corrected,
		ValidationStatus: status,
		ValidationReport: buildValidationReport(status, corrected, options.SchemaName, true),
	}, nil
}

func processProviderJSON(ctx context.Context, raw string, schema []byte, options ProcessOptions) (*Result, error) {
	extracted, err := ExtractProviderJSONResponse(raw)
	if err != nil {
		return nil, &ProcessError{Err: err, Recoverable: true}
	}

	markdown := strings.TrimSpace(extracted)
	if !options.RequireStructured {
		report := buildValidationReport("not_applicable", false, options.SchemaName, false)
		return &Result{
			Markdown:         markdown,
			ValidationStatus: "not_applicable",
			ValidationReport: report,
		}, nil
	}

	jsonText, err := ExtractJSON(markdown)
	if err != nil {
		return nil, &ProcessError{Err: err, Recoverable: true}
	}

	corrected := false
	if err := ValidateJSON(ctx, []byte(jsonText), schema); err != nil {
		if !errors.Is(err, ErrInvalidJSON) {
			return nil, &ProcessError{Err: err, Recoverable: false}
		}

		fixed := FixJSON(jsonText)
		if fixed == jsonText {
			return nil, &ProcessError{Err: err, Recoverable: true}
		}

		if validateErr := ValidateJSON(ctx, []byte(fixed), schema); validateErr != nil {
			recoverable := errors.Is(validateErr, ErrInvalidJSON)
			return nil, &ProcessError{Err: validateErr, Recoverable: recoverable}
		}

		jsonText = fixed
		corrected = true
	}

	status := "passed"
	if corrected {
		status = "corrected"
	}

	return &Result{
		Markdown:         markdown,
		JSON:             json.RawMessage(jsonText),
		Corrected:        corrected,
		ValidationStatus: status,
		ValidationReport: buildValidationReport(status, corrected, options.SchemaName, true),
	}, nil
}

func processJSONL(ctx context.Context, raw string, schema []byte, options ProcessOptions) (*Result, error) {
	extracted, err := ExtractJSONL(raw)
	if err != nil {
		return nil, err
	}
	extracted = strings.TrimSpace(extracted)

	// If structured output is not required, the extracted text is the markdown content.
	if !options.RequireStructured {
		report := buildValidationReport("not_applicable", false, options.SchemaName, false)
		return &Result{
			Markdown:         extracted,
			ValidationStatus: "not_applicable",
			ValidationReport: report,
		}, nil
	}

	jsonText, err := ExtractJSON(extracted)
	if err != nil {
		return nil, &ProcessError{Err: err, Recoverable: true}
	}

	// Structured path: the extracted content must contain valid JSON.
	if err := ValidateJSON(ctx, []byte(jsonText), schema); err != nil {
		if !errors.Is(err, ErrInvalidJSON) {
			return nil, &ProcessError{Err: err, Recoverable: false}
		}
		fixed := FixJSON(jsonText)
		if fixed == jsonText {
			return nil, &ProcessError{Err: err, Recoverable: true}
		}
		if validateErr := ValidateJSON(ctx, []byte(fixed), schema); validateErr != nil {
			recoverable := errors.Is(validateErr, ErrInvalidJSON)
			return nil, &ProcessError{Err: validateErr, Recoverable: recoverable}
		}
		return &Result{
			Markdown:         extracted,
			JSON:             json.RawMessage(fixed),
			Corrected:        true,
			ValidationStatus: "corrected",
			ValidationReport: buildValidationReport("corrected", true, options.SchemaName, true),
		}, nil
	}

	return &Result{
		Markdown:         extracted,
		JSON:             json.RawMessage(jsonText),
		ValidationStatus: "passed",
		ValidationReport: buildValidationReport("passed", false, options.SchemaName, true),
	}, nil
}

func buildValidationReport(status string, corrected bool, schemaName string, structured bool) json.RawMessage {
	report := map[string]any{
		"validation_status": status,
		"corrected":         corrected,
		"schema_name":       schemaName,
		"structured":        structured,
	}

	data, err := json.Marshal(report)
	if err != nil {
		return json.RawMessage(`{"validation_status":"failed"}`)
	}

	return data
}
