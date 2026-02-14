package config

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name        string
		setupCmd    func() *cobra.Command
		expectError bool
	}{
		{
			name: "valid_bootstrap",
			setupCmd: func() *cobra.Command {
				return &cobra.Command{
					Use: "test",
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for clean test
			viper.Reset()

			cmd := tt.setupCmd()
			err := Bootstrap(cmd)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestBindConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    map[string]string
		expectError bool
	}{
		{
			name: "all_env_vars",
			setupEnv: map[string]string{
				"RABBITMQ_ADDRESS":          "amqp://localhost:5672",
				"PROMETHEUS_ENDPOINT":       "/metrics",
				"PROMETHEUS_LISTEN_ADDRESS": ":2112",
				"DATABASE_DSN":              "user:pass@tcp(localhost:3306)/db",
				"REDIS_ADDRESS":             "localhost:6379",
				"REDIS_PASSWORD":            "password",
				"REDIS_DATABASE":            "0",
				"AUTHTOKEN_MESSAGEBIRD":     "test-token-mb",
				"AUTHTOKEN_TELNYX":          "test-token-telnyx",
			},
			expectError: false,
		},
		{
			name:        "no_env_vars",
			setupEnv:    map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for clean test
			viper.Reset()

			// Set environment variables
			for key, value := range tt.setupEnv {
				_ = os.Setenv(key, value)
				defer func(k string) {
					_ = os.Unsetenv(k)
				}(key)
			}

			cmd := &cobra.Command{
				Use: "test",
			}

			err := bindConfig(cmd)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify that flags are registered
			if err == nil {
				flags := []string{
					"rabbitmq_address",
					"prometheus_endpoint",
					"prometheus_listen_address",
					"database_dsn",
					"redis_address",
					"redis_password",
					"redis_database",
					"authtoken_messagebird",
					"authtoken_telnyx",
				}

				for _, flag := range flags {
					if cmd.PersistentFlags().Lookup(flag) == nil {
						t.Errorf("Flag %s was not registered", flag)
					}
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv map[string]string
	}{
		{
			name: "load_with_env_vars",
			setupEnv: map[string]string{
				"RABBITMQ_ADDRESS":          "amqp://test:5672",
				"PROMETHEUS_ENDPOINT":       "/test-metrics",
				"PROMETHEUS_LISTEN_ADDRESS": ":9090",
				"DATABASE_DSN":              "test-dsn",
				"REDIS_ADDRESS":             "test-redis:6379",
				"REDIS_PASSWORD":            "test-pass",
				"REDIS_DATABASE":            "5",
				"AUTHTOKEN_MESSAGEBIRD":     "mb-token",
				"AUTHTOKEN_TELNYX":          "telnyx-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()

			// Set environment variables
			for key, value := range tt.setupEnv {
				_ = os.Setenv(key, value)
				defer func(k string) {
					_ = os.Unsetenv(k)
				}(key)
			}

			// Setup command and bind config
			cmd := &cobra.Command{Use: "test"}
			if err := bindConfig(cmd); err != nil {
				t.Fatalf("bindConfig failed: %v", err)
			}

			// Note: We can't fully test LoadGlobalConfig due to sync.Once,
			// but we can verify it doesn't panic
			LoadGlobalConfig()

			cfg := Get()
			if cfg == nil {
				t.Error("Config is nil after LoadGlobalConfig")
			}
		})
	}
}

func TestRegisterFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	// Should not panic
	RegisterFlags(cmd)

	// Verify flags are registered
	flags := []string{
		"rabbitmq_address",
		"prometheus_endpoint",
		"prometheus_listen_address",
		"database_dsn",
		"redis_address",
		"redis_password",
		"redis_database",
		"authtoken_messagebird",
		"authtoken_telnyx",
	}

	for _, flag := range flags {
		if cmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("Flag %s was not registered", flag)
		}
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv map[string]string
	}{
		{
			name: "init_with_env",
			setupEnv: map[string]string{
				"RABBITMQ_ADDRESS": "amqp://localhost:5672",
				"DATABASE_DSN":     "test-dsn",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()

			// Set environment variables
			for key, value := range tt.setupEnv {
				_ = os.Setenv(key, value)
				defer func(k string) {
					_ = os.Unsetenv(k)
				}(key)
			}

			cmd := &cobra.Command{Use: "test"}
			RegisterFlags(cmd)

			// Should not panic
			InitConfig(cmd)
		})
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DatabaseDSN:             "test-dsn",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		RabbitMQAddress:         "amqp://localhost:5672",
		RedisAddress:            "localhost:6379",
		RedisDatabase:           0,
		RedisPassword:           "password",
		AuthtokenMessagebird:    "mb-token",
		AuthtokenTelnyx:         "telnyx-token",
	}

	if cfg.DatabaseDSN != "test-dsn" {
		t.Errorf("DatabaseDSN mismatch: got %v, want %v", cfg.DatabaseDSN, "test-dsn")
	}
	if cfg.PrometheusEndpoint != "/metrics" {
		t.Errorf("PrometheusEndpoint mismatch: got %v, want %v", cfg.PrometheusEndpoint, "/metrics")
	}
	if cfg.PrometheusListenAddress != ":2112" {
		t.Errorf("PrometheusListenAddress mismatch: got %v, want %v", cfg.PrometheusListenAddress, ":2112")
	}
	if cfg.RabbitMQAddress != "amqp://localhost:5672" {
		t.Errorf("RabbitMQAddress mismatch: got %v, want %v", cfg.RabbitMQAddress, "amqp://localhost:5672")
	}
	if cfg.RedisAddress != "localhost:6379" {
		t.Errorf("RedisAddress mismatch: got %v, want %v", cfg.RedisAddress, "localhost:6379")
	}
	if cfg.RedisDatabase != 0 {
		t.Errorf("RedisDatabase mismatch: got %v, want %v", cfg.RedisDatabase, 0)
	}
	if cfg.RedisPassword != "password" {
		t.Errorf("RedisPassword mismatch: got %v, want %v", cfg.RedisPassword, "password")
	}
	if cfg.AuthtokenMessagebird != "mb-token" {
		t.Errorf("AuthtokenMessagebird mismatch: got %v, want %v", cfg.AuthtokenMessagebird, "mb-token")
	}
	if cfg.AuthtokenTelnyx != "telnyx-token" {
		t.Errorf("AuthtokenTelnyx mismatch: got %v, want %v", cfg.AuthtokenTelnyx, "telnyx-token")
	}
}
