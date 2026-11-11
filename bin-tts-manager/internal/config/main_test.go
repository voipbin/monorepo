package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				AWSAccessKey:            "AKIA...",
				AWSSecretKey:            "secret...",
				ElevenlabsAPIKey:        "elevenlabs-api-key",
				GCPTTSEndpoint:          "texttospeech.googleapis.com:443",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = tt.setupConfig

			res := Get()

			if res.RabbitMQAddress != tt.setupConfig.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.setupConfig.RabbitMQAddress, res.RabbitMQAddress)
			}
			if res.PrometheusEndpoint != tt.setupConfig.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.setupConfig.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.setupConfig.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.setupConfig.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.AWSAccessKey != tt.setupConfig.AWSAccessKey {
				t.Errorf("Wrong AWSAccessKey. expect: %s, got: %s", tt.setupConfig.AWSAccessKey, res.AWSAccessKey)
			}
			if res.AWSSecretKey != tt.setupConfig.AWSSecretKey {
				t.Errorf("Wrong AWSSecretKey. expect: %s, got: %s", tt.setupConfig.AWSSecretKey, res.AWSSecretKey)
			}
			if res.ElevenlabsAPIKey != tt.setupConfig.ElevenlabsAPIKey {
				t.Errorf("Wrong ElevenlabsAPIKey. expect: %s, got: %s", tt.setupConfig.ElevenlabsAPIKey, res.ElevenlabsAPIKey)
			}
			if res.GCPTTSEndpoint != tt.setupConfig.GCPTTSEndpoint {
				t.Errorf("Wrong GCPTTSEndpoint. expect: %s, got: %s", tt.setupConfig.GCPTTSEndpoint, res.GCPTTSEndpoint)
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name string

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			err := Bootstrap(rootCmd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify flags were registered
			if rootCmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
				t.Errorf("Expected prometheus_endpoint flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
				t.Errorf("Expected prometheus_listen_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("aws_access_key") == nil {
				t.Errorf("Expected aws_access_key flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("aws_secret_key") == nil {
				t.Errorf("Expected aws_secret_key flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("elevenlabs_api_key") == nil {
				t.Errorf("Expected elevenlabs_api_key flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("gcp_tts_endpoint") == nil {
				t.Errorf("Expected gcp_tts_endpoint flag to be registered")
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loads_global_config_only_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once for test isolation
			once = sync.Once{}

			// First call should work without panic
			LoadGlobalConfig()

			// Second call should be a no-op due to sync.Once
			LoadGlobalConfig()

			// Verify Get returns a valid pointer
			cfg := Get()
			if cfg == nil {
				t.Errorf("Expected non-nil config from Get()")
			}
		})
	}
}

func Test_EnvValue(t *testing.T) {
	c := &Config{AWSAccessKey: "access-test", ElevenlabsAPIKey: "", ConfigEncryptionKeys: "1:x", AWSSecretKey: "secret-test"}

	tests := []struct {
		name string

		envName string

		expectValue string
		expectOK    bool
	}{
		{name: "aws access key", envName: "AWS_ACCESS_KEY", expectValue: "access-test", expectOK: true},
		{name: "aws secret key", envName: "AWS_SECRET_KEY", expectValue: "secret-test", expectOK: true},
		{name: "empty value is not set", envName: "ELEVENLABS_API_KEY", expectValue: "", expectOK: false},
		{name: "the master key list is not served to the registry", envName: "TTS_CONFIG_ENCRYPTION_KEYS", expectOK: false},
		{name: "unknown name", envName: "DATABASE_DSN", expectOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := c.EnvValue(tt.envName)
			if v != tt.expectValue || ok != tt.expectOK {
				t.Errorf("Wrong match. expect: %q %v, got: %q %v", tt.expectValue, tt.expectOK, v, ok)
			}
		})
	}
}

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
	t.Setenv("TTS_CONFIG_ENCRYPTION_KEYS", "1:prefixed-value")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "1:prefixed-value" {
		t.Errorf("Wrong match. expect: the prefixed value, got: %q", got)
	}
	for _, name := range []string{"tts_config_encryption_keys", "config_encryption_keys"} {
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
