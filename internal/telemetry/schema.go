package telemetry

import (
	"errors"
	"fmt"
	"strings"
)

const ProviderMetricNamespacePrefix = "provider."

var ErrProviderMetricLeakage = errors.New("metric key violates common/provider namespace schema")

type Schema struct {
	CommonMetrics []string
}

func CommonSchema() Schema {
	return Schema{CommonMetrics: RF44MetricNames()}
}

func (s Schema) IsCommonMetric(name string) bool {
	for _, m := range s.CommonMetrics {
		if m == name {
			return true
		}
	}
	return false
}

func (s Schema) ValidateKey(key string) error {
	if s.IsCommonMetric(key) {
		return nil
	}
	if strings.HasPrefix(key, ProviderMetricNamespacePrefix) {
		rest := strings.TrimPrefix(key, ProviderMetricNamespacePrefix)
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("%w: %q must follow provider.<name>.<key>", ErrProviderMetricLeakage, key)
		}
		return nil
	}
	return fmt.Errorf("%w: %q is neither a common metric nor namespaced under %q",
		ErrProviderMetricLeakage, key, ProviderMetricNamespacePrefix)
}

func (s Schema) ValidateNoLeakage(keys []string) error {
	for _, key := range keys {
		if err := s.ValidateKey(key); err != nil {
			return err
		}
	}
	return nil
}
