package components

import (
	"charm.land/huh/v2"
)

// InputField describes a single field in an interactive workflow input form.
type InputField struct {
	Name        string   // map key used to retrieve the collected value
	Type        string   // "text" | "multiline" | "select" | "confirm"
	Label       string   // displayed title / prompt
	Placeholder string   // hint shown when empty (text only)
	Options     []string // choices for select type
}

// BuildResult holds collected values after a form is submitted.
// StringValues covers text, multiline and select fields.
// BoolValues covers confirm fields.
type BuildResult struct {
	StringValues map[string]*string
	BoolValues   map[string]*bool
}

// BuildInputForm creates a huh.Form for the given fields.
// It returns the form and a BuildResult whose pointers are populated when the
// form completes. Callers must call form.Run() (or embed it in a Bubbletea
// program) before reading the values.
func BuildInputForm(fields []InputField) (*huh.Form, BuildResult) {
	result := BuildResult{
		StringValues: make(map[string]*string, len(fields)),
		BoolValues:   make(map[string]*bool),
	}

	huhFields := make([]huh.Field, 0, len(fields))
	for _, f := range fields {
		switch f.Type {
		case "multiline":
			val := new(string)
			result.StringValues[f.Name] = val
			huhFields = append(huhFields,
				huh.NewText().
					Title(f.Label).
					Value(val),
			)
		case "select":
			val := new(string)
			result.StringValues[f.Name] = val
			opts := make([]huh.Option[string], len(f.Options))
			for i, o := range f.Options {
				opts[i] = huh.NewOption(o, o)
			}
			huhFields = append(huhFields,
				huh.NewSelect[string]().
					Title(f.Label).
					Options(opts...).
					Value(val),
			)
		case "confirm":
			val := new(bool)
			result.BoolValues[f.Name] = val
			huhFields = append(huhFields,
				huh.NewConfirm().
					Title(f.Label).
					Value(val),
			)
		default: // "text"
			val := new(string)
			result.StringValues[f.Name] = val
			huhFields = append(huhFields,
				huh.NewInput().
					Title(f.Label).
					Placeholder(f.Placeholder).
					Value(val),
			)
		}
	}

	form := huh.NewForm(huh.NewGroup(huhFields...))
	return form, result
}
