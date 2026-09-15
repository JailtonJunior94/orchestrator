package specs

import (
	"slices"
	"strings"
)

type EnvPolicy struct {
	stripVars []string
}

func (c *Catalog) NewEnvPolicy(stripVars ...string) EnvPolicy {
	return EnvPolicy{stripVars: slices.Clone(stripVars)}
}

func (p EnvPolicy) StripVars() []string {
	return slices.Clone(p.stripVars)
}

func (p EnvPolicy) IsZero() bool {
	return len(p.stripVars) == 0
}

func (p EnvPolicy) Apply(environ []string) []string {
	if p.IsZero() {
		return nil
	}
	strip := make(map[string]bool, len(p.stripVars))
	for _, v := range p.stripVars {
		strip[v] = true
	}
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if strip[name] {
			continue
		}
		out = append(out, kv)
	}
	return out
}
