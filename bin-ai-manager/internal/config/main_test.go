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
				DatabaseDSN:             "user:pass@tcp(127.0.0.1:3306)/db",
				RedisAddress:            "127.0.0.1:6379",
				RedisPassword:           "secret",
				RedisDatabase:           1,
				EngineKeyChatGPT:        "sk-test-key",
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
			if res.DatabaseDSN != tt.setupConfig.DatabaseDSN {
				t.Errorf("Wrong DatabaseDSN. expect: %s, got: %s", tt.setupConfig.DatabaseDSN, res.DatabaseDSN)
			}
			if res.RedisAddress != tt.setupConfig.RedisAddress {
				t.Errorf("Wrong RedisAddress. expect: %s, got: %s", tt.setupConfig.RedisAddress, res.RedisAddress)
			}
			if res.RedisPassword != tt.setupConfig.RedisPassword {
				t.Errorf("Wrong RedisPassword. expect: %s, got: %s", tt.setupConfig.RedisPassword, res.RedisPassword)
			}
			if res.RedisDatabase != tt.setupConfig.RedisDatabase {
				t.Errorf("Wrong RedisDatabase. expect: %d, got: %d", tt.setupConfig.RedisDatabase, res.RedisDatabase)
			}
			if res.EngineKeyChatGPT != tt.setupConfig.EngineKeyChatGPT {
				t.Errorf("Wrong EngineKeyChatGPT. expect: %s, got: %s", tt.setupConfig.EngineKeyChatGPT, res.EngineKeyChatGPT)
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
			if rootCmd.PersistentFlags().Lookup("database_dsn") == nil {
				t.Errorf("Expected database_dsn flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_address") == nil {
				t.Errorf("Expected redis_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_password") == nil {
				t.Errorf("Expected redis_password flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_database") == nil {
				t.Errorf("Expected redis_database flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("engine_key_chatgpt") == nil {
				t.Errorf("Expected engine_key_chatgpt flag to be registered")
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
