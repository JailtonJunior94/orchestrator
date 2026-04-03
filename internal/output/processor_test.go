package output

import (
	"context"
	"errors"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "json code block",
			input: "text\n```json\n{\"ok\":true}\n```",
			want:  "{\"ok\":true}",
		},
		{
			name:  "inline json",
			input: "prefix {\"ok\":true} suffix",
			want:  "{\"ok\":true}",
		},
		{
			name:    "missing json",
			input:   "just markdown",
			wantErr: ErrJSONNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExtractJSON(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got = %q want %q", got, tt.want)
			}
		})
	}
}

func TestFixJSON(t *testing.T) {
	t.Parallel()

	got := FixJSON("{'a': 'b',}")
	if got != "{\"a\": \"b\"}" {
		t.Fatalf("fixed json = %q", got)
	}
}

func TestValidateJSON(t *testing.T) {
	t.Parallel()

	schema := []byte(`{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
	if err := ValidateJSON(context.Background(), []byte(`{"name":"orq"}`), schema); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := ValidateJSON(context.Background(), []byte(`{"age":1}`), schema)
	if !errors.Is(err, ErrSchemaValidation) {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestProcessor(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	result, err := processor.Process(context.Background(), "```json\n{'name':'orq',}\n```", nil, ProcessOptions{RequireStructured: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Corrected {
		t.Fatal("expected corrected result")
	}

	result, err = processor.Process(context.Background(), "markdown only", nil, ProcessOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationStatus != "not_applicable" {
		t.Fatalf("validation status = %q", result.ValidationStatus)
	}

	_, err = processor.Process(context.Background(), "markdown only", nil, ProcessOptions{RequireStructured: true})
	if err == nil || !IsRecoverable(err) {
		t.Fatalf("expected recoverable error, got %v", err)
	}
}
