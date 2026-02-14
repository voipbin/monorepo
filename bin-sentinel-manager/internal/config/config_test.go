package config

import (
	"os"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
		expectCfg   Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
			expectCfg:   Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
			},
			expectCfg: Config{
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the global config
			cfg = tt.setupConfig

			res := Get()

			if res.PrometheusEndpoint != tt.expectCfg.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.expectCfg.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.expectCfg.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.expectCfg.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.expectCfg.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.expectCfg.RabbitMQAddress, res.RabbitMQAddress)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name string

		prometheusEndpoint      string
		prometheusListenAddress string
		rabbitmqAddress         string

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			prometheusEndpoint:      "/metrics",
			prometheusListenAddress: ":2112",
			rabbitmqAddress:         "amqp://guest:guest@localhost:5672",

			expectErr: false,
		},
		{
			name: "initializes_with_custom_values",

			prometheusEndpoint:      "/custom-metrics",
			prometheusListenAddress: ":9090",
			rabbitmqAddress:         "amqp://user:pass@rabbitmq.example.com:5672",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("prometheus_endpoint", tt.prometheusEndpoint, "")
			cmd.Flags().String("prometheus_listen_address", tt.prometheusListenAddress, "")
			cmd.Flags().String("rabbitmq_address", tt.rabbitmqAddress, "")

			err := InitConfig(cmd)

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

			res := Get()
			if res.PrometheusEndpoint != tt.prometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.prometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.prometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.prometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.rabbitmqAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.rabbitmqAddress, res.RabbitMQAddress)
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
			name: "bootstrap_with_default_flags",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper to clean state
			viper.Reset()

			cmd := &cobra.Command{}

			err := Bootstrap(cmd)

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

			// Verify flags were created
			if cmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}
			if cmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
				t.Errorf("Expected prometheus_endpoint flag to be registered")
			}
			if cmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
				t.Errorf("Expected prometheus_listen_address flag to be registered")
			}
		})
	}
}

func TestBootstrapWithEnv(t *testing.T) {
	tests := []struct {
		name string

		envVars map[string]string

		expectErr bool
	}{
		{
			name: "bootstrap_reads_env_vars",

			envVars: map[string]string{
				"RABBITMQ_ADDRESS":          "amqp://test:test@test:5672",
				"PROMETHEUS_ENDPOINT":       "/test-metrics",
				"PROMETHEUS_LISTEN_ADDRESS": ":9999",
			},

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper to clean state
			viper.Reset()

			// Set environment variables
			for k, v := range tt.envVars {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set env var %s: %v", k, err)
				}
				defer func(key string) {
					if err := os.Unsetenv(key); err != nil {
						t.Logf("Failed to unset env var %s: %v", key, err)
					}
				}(k)
			}

			cmd := &cobra.Command{}

			err := Bootstrap(cmd)

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

			// Verify bindings work by checking viper can read the env vars
			if viper.GetString("rabbitmq_address") != tt.envVars["RABBITMQ_ADDRESS"] {
				t.Errorf("Wrong rabbitmq_address. expect: %s, got: %s", tt.envVars["RABBITMQ_ADDRESS"], viper.GetString("rabbitmq_address"))
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string

		viperValues map[string]string
		expectCfg   Config
	}{
		{
			name: "loads_config_from_viper",

			viperValues: map[string]string{
				"prometheus_endpoint":       "/test-metrics",
				"prometheus_listen_address": ":8888",
				"rabbitmq_address":          "amqp://test:test@localhost:5672",
			},
			expectCfg: Config{
				PrometheusEndpoint:      "/test-metrics",
				PrometheusListenAddress: ":8888",
				RabbitMQAddress:         "amqp://test:test@localhost:5672",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper and once to clean state
			viper.Reset()
			once = sync.Once{}
			cfg = Config{}

			// Set viper values
			for k, v := range tt.viperValues {
				viper.Set(k, v)
			}

			LoadGlobalConfig()

			res := Get()
			if res.PrometheusEndpoint != tt.expectCfg.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.expectCfg.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.expectCfg.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.expectCfg.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.expectCfg.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.expectCfg.RabbitMQAddress, res.RabbitMQAddress)
			}
		})
	}
}

func TestLoadGlobalConfigOnlyOnce(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loads_config_only_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper and once to clean state
			viper.Reset()
			once = sync.Once{}
			cfg = Config{}

			// Set initial viper values
			viper.Set("prometheus_endpoint", "/first")
			viper.Set("prometheus_listen_address", ":1111")
			viper.Set("rabbitmq_address", "amqp://first")

			LoadGlobalConfig()

			firstCfg := Get()

			// Change viper values
			viper.Set("prometheus_endpoint", "/second")
			viper.Set("prometheus_listen_address", ":2222")
			viper.Set("rabbitmq_address", "amqp://second")

			// Call LoadGlobalConfig again - should not change cfg
			LoadGlobalConfig()

			secondCfg := Get()

			// Verify config did not change (once.Do ensures it runs only once)
			if firstCfg.PrometheusEndpoint != secondCfg.PrometheusEndpoint {
				t.Errorf("Config should not change after second LoadGlobalConfig call")
			}
			if secondCfg.PrometheusEndpoint != "/first" {
				t.Errorf("Expected PrometheusEndpoint to be '/first', got: %s", secondCfg.PrometheusEndpoint)
			}
		})
	}
}

func TestInitConfigWithMissingFlags(t *testing.T) {
	tests := []struct {
		name string

		skipFlag  string
		expectErr bool
	}{
		{
			name: "error_when_prometheus_endpoint_flag_missing",

			skipFlag:  "prometheus_endpoint",
			expectErr: true,
		},
		{
			name: "error_when_prometheus_listen_address_flag_missing",

			skipFlag:  "prometheus_listen_address",
			expectErr: true,
		},
		{
			name: "error_when_rabbitmq_address_flag_missing",

			skipFlag:  "rabbitmq_address",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			cmd := &cobra.Command{}

			// Add flags except the one we're testing
			if tt.skipFlag != "prometheus_endpoint" {
				cmd.Flags().String("prometheus_endpoint", "/metrics", "")
			}
			if tt.skipFlag != "prometheus_listen_address" {
				cmd.Flags().String("prometheus_listen_address", ":2112", "")
			}
			if tt.skipFlag != "rabbitmq_address" {
				cmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "")
			}

			err := InitConfig(cmd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
