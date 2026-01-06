package streaminghandler

import (
	"fmt"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
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

// initProviders validates and initializes the provider priority list.
// Returns the validated list of providers or an error if validation fails.
func initProviders(priorityList []string, gcpClient *speech.Client, awsClient *transcribestreaming.Client) ([]STTProvider, error) {
	var validatedProviders []STTProvider

	for _, providerStr := range priorityList {
		provider, err := validateProvider(providerStr)
		if err != nil {
			return nil, fmt.Errorf("could not validate STT provider. provider: %s, err: %w", providerStr, err)
		}

		// Validate provider is initialized
		if provider == STTProviderGCP && gcpClient == nil {
			return nil, fmt.Errorf("STT provider '%s' listed in priority but not initialized (check GCP credentials)", STTProviderGCP)
		}
		if provider == STTProviderAWS && awsClient == nil {
			return nil, fmt.Errorf("STT provider '%s' listed in priority but not initialized (check AWS credentials)", STTProviderAWS)
		}

		// Check for duplicates (configuration error)
		for _, existing := range validatedProviders {
			if existing == provider {
				return nil, fmt.Errorf("duplicate STT provider '%s' in priority list. Please remove duplicates from configuration", provider)
			}
		}

		validatedProviders = append(validatedProviders, provider)
	}

	if len(validatedProviders) == 0 {
		return nil, fmt.Errorf("no valid STT providers in priority list")
	}

	return validatedProviders, nil
}
