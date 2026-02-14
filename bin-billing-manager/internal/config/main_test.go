package config

import (
	"testing"

	"github.com/spf13/cobra"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestInitPrometheus_DisabledWhenNotConfigured(t *testing.T) {
	// Reset global config to empty state
	globalConfig = Config{
		PrometheusEndpoint:      "",
		PrometheusListenAddress: "",
	}

	// Should not panic when endpoint/listen address not configured
	InitPrometheus()
}

func TestInitPrometheus_DisabledWhenPartiallyConfigured(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		listen   string
	}{
		{"missing endpoint", "", ":8080"},
		{"missing listen address", "/metrics", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = Config{
				PrometheusEndpoint:      tt.endpoint,
				PrometheusListenAddress: tt.listen,
			}

			// Should not panic when partially configured
			InitPrometheus()
		})
	}
}
