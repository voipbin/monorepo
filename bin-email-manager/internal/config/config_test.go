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
				SendgridAPIKey:          "SG.test-key",
				MailgunAPIKey:           "mailgun-test-key",
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
			if res.SendgridAPIKey != tt.setupConfig.SendgridAPIKey {
				t.Errorf("Wrong SendgridAPIKey. expect: %s, got: %s", tt.setupConfig.SendgridAPIKey, res.SendgridAPIKey)
			}
			if res.MailgunAPIKey != tt.setupConfig.MailgunAPIKey {
				t.Errorf("Wrong MailgunAPIKey. expect: %s, got: %s", tt.setupConfig.MailgunAPIKey, res.MailgunAPIKey)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
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
			rootCmd.Flags().String("database_dsn", "test-dsn", "")
			rootCmd.Flags().String("prometheus_endpoint", "/metrics", "")
			rootCmd.Flags().String("prometheus_listen_address", ":2112", "")
			rootCmd.Flags().String("rabbitmq_address", "amqp://localhost", "")
			rootCmd.Flags().String("redis_address", "localhost:6379", "")
			rootCmd.Flags().Int("redis_database", 1, "")
			rootCmd.Flags().String("redis_password", "", "")
			rootCmd.Flags().String("sendgrid_api_key", "SG.test", "")
			rootCmd.Flags().String("mailgun_api_key", "mg-test", "")

			// First call should initialize
			InitConfig(rootCmd)

			// Second call should be a no-op due to sync.Once
			InitConfig(rootCmd)

			// Should not panic
			res := Get()
			if res == nil {
				t.Errorf("Expected non-nil config from Get()")
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "bootstrap_initializes_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			err := Bootstrap(rootCmd)

			if err != nil {
				t.Errorf("Unexpected error from Bootstrap: %v", err)
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loads_global_config_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset for test isolation
			once = sync.Once{}
			globalConfig = Config{}

			// Set environment variables for testing
			t.Setenv("RABBITMQ_ADDRESS", "amqp://test-rabbitmq:5672")
			t.Setenv("DATABASE_DSN", "test-db-dsn")
			t.Setenv("REDIS_ADDRESS", "test-redis:6379")

			// Call LoadGlobalConfig
			LoadGlobalConfig()

			// Call again to test sync.Once
			LoadGlobalConfig()

			// Verify config loaded
			cfg := Get()
			if cfg.RabbitMQAddress != "amqp://test-rabbitmq:5672" {
				t.Errorf("Wrong RabbitMQAddress. expect: amqp://test-rabbitmq:5672, got: %s", cfg.RabbitMQAddress)
			}
		})
	}
}

func TestRegisterFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "registers_flags_successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			// This should not panic
			RegisterFlags(rootCmd)

			// Verify flags are registered
			flag := rootCmd.PersistentFlags().Lookup("rabbitmq_address")
			if flag == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}

			flag = rootCmd.PersistentFlags().Lookup("database_dsn")
			if flag == nil {
				t.Errorf("Expected database_dsn flag to be registered")
			}
		})
	}
}
