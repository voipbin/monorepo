package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil, expected *Config")
	}
}

func TestBootstrap(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	err := Bootstrap(cmd)
	if err != nil {
		t.Errorf("Bootstrap() returned error: %v", err)
	}

	flags := cmd.PersistentFlags()

	tests := []struct {
		name     string
		flagName string
	}{
		{"rabbitmq_address", "rabbitmq_address"},
		{"prometheus_endpoint", "prometheus_endpoint"},
		{"prometheus_listen_address", "prometheus_listen_address"},
		{"database_dsn", "database_dsn"},
		{"redis_address", "redis_address"},
		{"redis_password", "redis_password"},
		{"redis_database", "redis_database"},
		{"homer_api_address", "homer_api_address"},
		{"homer_auth_token", "homer_auth_token"},
		{"homer_whitelist", "homer_whitelist"},
		{"recovery_enabled", "recovery_enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := flags.Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %s was not registered", tt.flagName)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		RabbitMQAddress:         "amqp://localhost:5672",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":8080",
		DatabaseDSN:             "user:pass@tcp(localhost:3306)/db",
		RedisAddress:            "localhost:6379",
		RedisPassword:           "secret",
		RedisDatabase:           1,
		HomerAPIAddress:         "http://homer:9080",
		HomerAuthToken:          "test-token",
		HomerWhitelist:          []string{"192.168.1.1", "10.0.0.1"},
		RecoveryEnabled:         true,
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"RabbitMQAddress", cfg.RabbitMQAddress, "amqp://localhost:5672"},
		{"PrometheusEndpoint", cfg.PrometheusEndpoint, "/metrics"},
		{"PrometheusListenAddress", cfg.PrometheusListenAddress, ":8080"},
		{"DatabaseDSN", cfg.DatabaseDSN, "user:pass@tcp(localhost:3306)/db"},
		{"RedisAddress", cfg.RedisAddress, "localhost:6379"},
		{"RedisPassword", cfg.RedisPassword, "secret"},
		{"RedisDatabase", cfg.RedisDatabase, 1},
		{"HomerAPIAddress", cfg.HomerAPIAddress, "http://homer:9080"},
		{"HomerAuthToken", cfg.HomerAuthToken, "test-token"},
		{"HomerWhitelistLength", len(cfg.HomerWhitelist), 2},
		{"RecoveryEnabled", cfg.RecoveryEnabled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// Test_Bootstrap_recoveryEnabledDefault pins VOIP-1553: call recovery must be off unless
// RECOVERY_ENABLED is explicitly set to a true value.
func Test_Bootstrap_recoveryEnabledDefault(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	if err := Bootstrap(cmd); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	flag := cmd.PersistentFlags().Lookup("recovery_enabled")
	if flag == nil {
		t.Errorf("Wrong match. expect: recovery_enabled flag registered, got: nil")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("Wrong match. expect: false, got: %v", flag.DefValue)
	}
}

// Test_Bootstrap_recoveryEnabledEnv checks that RECOVERY_ENABLED is bound to recovery_enabled, so
// recovery can be turned back on by the env var after VOIP-1556 (VOIP-1553).
func Test_Bootstrap_recoveryEnabledEnv(t *testing.T) {
	tests := []struct {
		name string

		env string

		expectRes bool
	}{
		{
			name:      "true",
			env:       "true",
			expectRes: true,
		},
		{
			name:      "one",
			env:       "1",
			expectRes: true,
		},
		{
			name:      "false",
			env:       "false",
			expectRes: false,
		},
		{
			name:      "empty",
			env:       "",
			expectRes: false,
		},
		{
			name:      "unparsable",
			env:       "yes",
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RECOVERY_ENABLED", tt.env)

			cmd := &cobra.Command{
				Use:   "test",
				Short: "Test command",
			}
			if err := Bootstrap(cmd); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if res := viper.GetBool("recovery_enabled"); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
