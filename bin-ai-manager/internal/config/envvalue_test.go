package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Test_masterKeyIsEnvOnly checks that the master key list is read from the prefixed variable only,
// that an unprefixed variable is ignored, and that no command line flag exists for it.
func Test_masterKeyIsEnvOnly(t *testing.T) {
	viper.Reset()
	ResetForTest()
	t.Cleanup(func() {
		viper.Reset()
		ResetForTest()
	})

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")
	t.Setenv("AI_CONFIG_ENCRYPTION_KEYS", "1:prefixed-value")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "1:prefixed-value" {
		t.Errorf("Wrong match. expect: the prefixed value, got: %q", got)
	}
	for _, name := range []string{"ai_config_encryption_keys", "config_encryption_keys"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}

func Test_masterKeyUnprefixedIgnored(t *testing.T) {
	viper.Reset()
	ResetForTest()
	t.Cleanup(func() {
		viper.Reset()
		ResetForTest()
	})

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q", got)
	}
}

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
