package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	RegisterFlags(cmd)

	flags := cmd.Flags()

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
	c := Config{
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
		{"RabbitMQAddress", c.RabbitMQAddress, "amqp://localhost:5672"},
		{"PrometheusEndpoint", c.PrometheusEndpoint, "/metrics"},
		{"PrometheusListenAddress", c.PrometheusListenAddress, ":8080"},
		{"DatabaseDSN", c.DatabaseDSN, "user:pass@tcp(localhost:3306)/db"},
		{"RedisAddress", c.RedisAddress, "localhost:6379"},
		{"RedisPassword", c.RedisPassword, "secret"},
		{"RedisDatabase", c.RedisDatabase, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant interface{}
		expected interface{}
	}{
		{"defaultDatabaseDSN", defaultDatabaseDSN, "testid:testpassword@tcp(127.0.0.1:3306)/test"},
		{"defaultPrometheusEndpoint", defaultPrometheusEndpoint, "/metrics"},
		{"defaultPrometheusListenAddress", defaultPrometheusListenAddress, ":2112"},
		{"defaultRabbitMQAddress", defaultRabbitMQAddress, "amqp://guest:guest@localhost:5672"},
		{"defaultRedisAddress", defaultRedisAddress, "127.0.0.1:6379"},
		{"defaultRedisDatabase", defaultRedisDatabase, 1},
		{"defaultRedisPassword", defaultRedisPassword, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %v, got: %v", tt.expected, tt.constant)
			}
		})
	}
}
