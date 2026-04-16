package output

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestProcessorJSONL(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	t.Run("valid jsonl returns content", func(t *testing.T) {
		t.Parallel()
		raw := `{"type":"message","content":"the answer"}` + "\n"
		result, err := processor.Process(context.Background(), raw, nil, ProcessOptions{
			OutputFormat: "jsonl",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Markdown != "the answer" {
			t.Fatalf("markdown = %q, want %q", result.Markdown, "the answer")
		}
	})

	t.Run("invalid jsonl returns recoverable error", func(t *testing.T) {
		t.Parallel()
		_, err := processor.Process(context.Background(), "not-json", nil, ProcessOptions{
			OutputFormat: "jsonl",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsRecoverable(err) {
			t.Fatalf("expected recoverable error, got %v", err)
		}
	})

	t.Run("structured jsonl extracts markdown and validates embedded json", func(t *testing.T) {
		t.Parallel()
		raw := "{\"type\":\"message\",\"content\":\"Summary\\n```json\\n{\\\"doc\\\":\\\"ok\\\"}\\n```\"}\n"
		result, err := processor.Process(context.Background(), raw, []byte(`{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`), ProcessOptions{
			RequireStructured: true,
			OutputFormat:      "jsonl",
			SchemaName:        "prd/v1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Markdown != "Summary\n```json\n{\"doc\":\"ok\"}\n```" {
			t.Fatalf("markdown = %q", result.Markdown)
		}
		if string(result.JSON) != `{"doc":"ok"}` {
			t.Fatalf("json = %s", string(result.JSON))
		}
	})

	t.Run("structured jsonl fixture matches documented codex stream", func(t *testing.T) {
		t.Parallel()

		raw, err := os.ReadFile(filepath.Join("testdata", "codex_completion.jsonl"))
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		result, err := processor.Process(context.Background(), string(raw), []byte(`{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`), ProcessOptions{
			RequireStructured: true,
			OutputFormat:      "jsonl",
			SchemaName:        "tasks/v1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result.JSON) != `{"doc":"fixture"}` {
			t.Fatalf("json = %s", string(result.JSON))
		}
		if result.ValidationStatus != "passed" {
			t.Fatalf("status = %q", result.ValidationStatus)
		}
	})

	t.Run("default flow unaffected when OutputFormat empty", func(t *testing.T) {
		t.Parallel()
		result, err := processor.Process(context.Background(), "plain text", nil, ProcessOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.ValidationStatus != "not_applicable" {
			t.Fatalf("status = %q", result.ValidationStatus)
		}
	})
}

func TestProcessorProviderJSON(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	t.Run("structured provider json extracts response markdown and schema payload", func(t *testing.T) {
		t.Parallel()

		raw := "{\"response\":\"Summary\\n\\n```json\\n{\\\"doc\\\":\\\"ok\\\"}\\n```\",\"stats\":{\"session\":{\"duration\":12}},\"error\":null}"
		result, err := processor.Process(context.Background(), raw, []byte(`{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`), ProcessOptions{
			RequireStructured: true,
			OutputFormat:      "json",
			SchemaName:        "prd/v1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Markdown != "Summary\n\n```json\n{\"doc\":\"ok\"}\n```" {
			t.Fatalf("markdown = %q", result.Markdown)
		}
		if string(result.JSON) != `{"doc":"ok"}` {
			t.Fatalf("json = %s", string(result.JSON))
		}
	})

	t.Run("structured claude json envelope extracts result markdown and schema payload", func(t *testing.T) {
		t.Parallel()

		raw := "{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":\"Summary\\n\\n```json\\n{\\\"doc\\\":\\\"ok\\\"}\\n```\"}"
		result, err := processor.Process(context.Background(), raw, []byte(`{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`), ProcessOptions{
			RequireStructured: true,
			OutputFormat:      "json",
			SchemaName:        "prd/v1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Markdown != "Summary\n\n```json\n{\"doc\":\"ok\"}\n```" {
			t.Fatalf("markdown = %q", result.Markdown)
		}
		if string(result.JSON) != `{"doc":"ok"}` {
			t.Fatalf("json = %s", string(result.JSON))
		}
	})

	t.Run("invalid provider json envelope is recoverable", func(t *testing.T) {
		t.Parallel()

		_, err := processor.Process(context.Background(), `{"stats":{},"error":null}`, nil, ProcessOptions{
			OutputFormat: "json",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsRecoverable(err) {
			t.Fatalf("expected recoverable error, got %v", err)
		}
	})
}
