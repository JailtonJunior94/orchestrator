package workflows

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// TemplateVars holds the variables available for template resolution.
type TemplateVars struct {
	Input       string
	StepOutputs map[string]string
}

// TemplateResolver resolves template variables in a string.
type TemplateResolver interface {
	Resolve(ctx context.Context, template string, vars TemplateVars) (string, error)
}

type templateResolver struct{}

// NewTemplateResolver creates a TemplateResolver.
func NewTemplateResolver() TemplateResolver {
	return &templateResolver{}
}

var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func (r *templateResolver) Resolve(_ context.Context, tmpl string, vars TemplateVars) (string, error) {
	var resolveErr error

	result := templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		if resolveErr != nil {
			return match
		}

		key := strings.TrimSpace(match[2 : len(match)-2])

		if key == "input" {
			return vars.Input
		}

		if strings.HasPrefix(key, "steps.") && strings.HasSuffix(key, ".output") {
			parts := strings.SplitN(key, ".", 3)
			if len(parts) == 3 {
				stepName := parts[1]
				if output, ok := vars.StepOutputs[stepName]; ok {
					return output
				}
				resolveErr = fmt.Errorf("%w: %q", ErrUnresolvedVariable, key)
				return match
			}
		}

		resolveErr = fmt.Errorf("%w: %q", ErrUnresolvedVariable, key)
		return match
	})

	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}
