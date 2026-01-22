package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
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
