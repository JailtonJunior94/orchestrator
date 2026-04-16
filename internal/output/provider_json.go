package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

type providerJSONEnvelope struct {
	Response *string `json:"response"`
	Result   *string `json:"result"`
	Error    any     `json:"error"`
}

// ExtractProviderJSONResponse extracts the textual model response from a provider
// envelope such as Gemini CLI's JSON output mode.
func ExtractProviderJSONResponse(raw string) (string, error) {
	var envelope providerJSONEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return "", fmt.Errorf("provider json: decode envelope: %w", err)
	}

	if envelope.Response != nil && strings.TrimSpace(*envelope.Response) != "" {
		return *envelope.Response, nil
	}

	if envelope.Result != nil && strings.TrimSpace(*envelope.Result) != "" {
		return *envelope.Result, nil
	}

	if envelope.Error != nil {
		return "", fmt.Errorf("provider json: response missing from envelope")
	}

	return "", fmt.Errorf("provider json: response field is empty")
}
