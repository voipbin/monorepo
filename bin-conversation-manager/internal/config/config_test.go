package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig *Config
		expectPanic bool
	}{
		{
			name: "returns_config_when_initialized",

			setupConfig: &Config{
				DatabaseDSN:             "user:pass@tcp(127.0.0.1:3306)/db",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				RedisAddress:            "127.0.0.1:6379",
				RedisDatabase:           1,
				RedisPassword:           "secret",
			},
			expectPanic: false,
		},
		{
			name: "panics_when_not_initialized",

			setupConfig: nil,
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = tt.setupConfig

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic but did not get one")
					}
				}()
			}

			res := Get()

			if tt.expectPanic {
				t.Errorf("Should have panicked")
				return
			}

			if res.DatabaseDSN != tt.setupConfig.DatabaseDSN {
				t.Errorf("Wrong DatabaseDSN. expect: %s, got: %s", tt.setupConfig.DatabaseDSN, res.DatabaseDSN)
			}
			if res.PrometheusEndpoint != tt.setupConfig.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.setupConfig.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.setupConfig.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.setupConfig.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.setupConfig.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.setupConfig.RabbitMQAddress, res.RabbitMQAddress)
			}
			if res.RedisAddress != tt.setupConfig.RedisAddress {
				t.Errorf("Wrong RedisAddress. expect: %s, got: %s", tt.setupConfig.RedisAddress, res.RedisAddress)
			}
			if res.RedisDatabase != tt.setupConfig.RedisDatabase {
				t.Errorf("Wrong RedisDatabase. expect: %d, got: %d", tt.setupConfig.RedisDatabase, res.RedisDatabase)
			}
			if res.RedisPassword != tt.setupConfig.RedisPassword {
				t.Errorf("Wrong RedisPassword. expect: %s, got: %s", tt.setupConfig.RedisPassword, res.RedisPassword)
			}
		})
	}
}

func TestRegisterFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "registers_all_flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			RegisterFlags(rootCmd)

			// Verify flags were registered
			if rootCmd.Flags().Lookup("rabbitmq_address") == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}
			if rootCmd.Flags().Lookup("prometheus_endpoint") == nil {
				t.Errorf("Expected prometheus_endpoint flag to be registered")
			}
			if rootCmd.Flags().Lookup("prometheus_listen_address") == nil {
				t.Errorf("Expected prometheus_listen_address flag to be registered")
			}
			if rootCmd.Flags().Lookup("database_dsn") == nil {
				t.Errorf("Expected database_dsn flag to be registered")
			}
			if rootCmd.Flags().Lookup("redis_address") == nil {
				t.Errorf("Expected redis_address flag to be registered")
			}
			if rootCmd.Flags().Lookup("redis_password") == nil {
				t.Errorf("Expected redis_password flag to be registered")
			}
			if rootCmd.Flags().Lookup("redis_database") == nil {
				t.Errorf("Expected redis_database flag to be registered")
			}
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "initializes_config_only_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once and cfg for test isolation
			once = sync.Once{}
			cfg = nil

			rootCmd := &cobra.Command{}
			RegisterFlags(rootCmd)

			// First call should initialize
			Init(rootCmd)

			// Verify cfg is initialized
			if cfg == nil {
				t.Errorf("Expected cfg to be initialized")
			}

			// Second call should be a no-op due to sync.Once
			Init(rootCmd)

			// Should not panic
			res := Get()
			if res == nil {
				t.Errorf("Expected non-nil config from Get()")
			}
		})
	}
}
