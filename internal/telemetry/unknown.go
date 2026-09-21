package telemetry

import (
	"errors"
	"fmt"
)

const UnknownMetricValue = "unknown"

var ErrUnidentifiedEstimationMethod = errors.New("metric value estimated without an identified calculation method")

func ResolveTokenMetric(value, calculationMethod string) (string, error) {
	if value == "" {
		return UnknownMetricValue, nil
	}
	if calculationMethod == "" {
		return "", fmt.Errorf("%w: value %q present without a calculation method", ErrUnidentifiedEstimationMethod, value)
	}
	return value, nil
}

func ResolveMetric(available bool, value string) string {
	if !available || value == "" {
		return UnknownMetricValue
	}
	return value
}
