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
			name: "returns_config_when_initialized",

			setupConfig: Config{
				DatabaseDSN:             "user:pass@tcp(127.0.0.1:3306)/db",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				RedisAddress:            "127.0.0.1:6379",
				RedisDatabase:           1,
				RedisPassword:           "secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = tt.setupConfig

			res := Get()

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
			// Reset once and globalConfig for test isolation
			once = sync.Once{}
			globalConfig = Config{}

			rootCmd := &cobra.Command{}
			RegisterFlags(rootCmd)

			// First call should initialize
			Init(rootCmd)

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
