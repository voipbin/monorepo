package streaminghandler

import (
	"fmt"
	"strings"
)

// validateProvider validates and normalizes a provider name (case-insensitive).
// Returns the canonical STTProvider constant or an error if invalid.
func validateProvider(raw string) (STTProvider, error) {
	normalized := strings.TrimSpace(raw)
	provider := STTProvider(strings.ToUpper(normalized))

	switch provider {
	case STTProviderGCP, STTProviderAWS:
		return provider, nil
	default:
		return "", fmt.Errorf("unknown STT provider: %s. Valid providers: %s, %s (case-insensitive)",
			raw, STTProviderGCP, STTProviderAWS)
	}
}
