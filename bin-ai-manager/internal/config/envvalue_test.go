package config

import (
	"testing"
)

// The public bounds of the Builder time budget are the ones that Validate enforces: 50 seconds of
// LLM deadline is accepted and 51 is refused. builderhandler guards the key check against them.
func Test_builderTimeBudgetBounds(t *testing.T) {
	if BuilderRPCTimeoutSeconds != 55 || BuilderLLMTimeoutMaxSeconds != 50 {
		t.Fatalf("Wrong match. expect: 55 and 50, got: %d and %d", BuilderRPCTimeoutSeconds, BuilderLLMTimeoutMaxSeconds)
	}

	old := globalConfig
	t.Cleanup(func() { globalConfig = old })

	globalConfig.AIBuilderMaxOutputTokens = 100
	globalConfig.AIBuilderDailyLimit = 10
	globalConfig.AIBuilderMaxConcurrent = 2
	globalConfig.AIBuilderModel = "model-x"

	for _, tt := range []struct {
		seconds   int
		expectErr bool
	}{{BuilderLLMTimeoutMaxSeconds, false}, {BuilderLLMTimeoutMaxSeconds + 1, true}} {
		globalConfig.AIBuilderLLMTimeoutSeconds = tt.seconds
		if err := validateBuilderConfig(); (err != nil) != tt.expectErr {
			t.Errorf("Wrong match. %d seconds expect error: %v, got: %v", tt.seconds, tt.expectErr, err)
		}
	}
}
