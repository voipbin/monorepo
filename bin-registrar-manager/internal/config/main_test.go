package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	cfg := &Config{
		RabbitMQAddress:         "amqp://localhost:5672",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":8080",
		DatabaseDSNBin:          "user:pass@tcp(localhost:3306)/bin",
		DatabaseDSNAsterisk:     "user:pass@tcp(localhost:3306)/asterisk",
		RedisAddress:            "localhost:6379",
		RedisPassword:           "secret",
		RedisDatabase:           0,
		DomainNameExtension:     "ext.example.com",
		DomainNameTrunk:         "trunk.example.com",
	}

	if cfg.RabbitMQAddress != "amqp://localhost:5672" {
		t.Errorf("Expected RabbitMQAddress 'amqp://localhost:5672', got '%s'", cfg.RabbitMQAddress)
	}
	if cfg.PrometheusEndpoint != "/metrics" {
		t.Errorf("Expected PrometheusEndpoint '/metrics', got '%s'", cfg.PrometheusEndpoint)
	}
	if cfg.DatabaseDSNBin != "user:pass@tcp(localhost:3306)/bin" {
		t.Errorf("Expected DatabaseDSNBin 'user:pass@tcp(localhost:3306)/bin', got '%s'", cfg.DatabaseDSNBin)
	}
	if cfg.RedisDatabase != 0 {
		t.Errorf("Expected RedisDatabase 0, got %d", cfg.RedisDatabase)
	}
}

func TestBindConfig(t *testing.T) {
	// Reset viper state
	viper.Reset()

	cmd := &cobra.Command{}
	err := bindConfig(cmd)
	if err != nil {
		t.Errorf("bindConfig() failed: %v", err)
	}

	// Verify flags were registered
	flags := cmd.PersistentFlags()
	if flags.Lookup("rabbitmq_address") == nil {
		t.Error("rabbitmq_address flag not registered")
	}
	if flags.Lookup("prometheus_endpoint") == nil {
		t.Error("prometheus_endpoint flag not registered")
	}
	if flags.Lookup("database_dsn_bin") == nil {
		t.Error("database_dsn_bin flag not registered")
	}
	if flags.Lookup("redis_address") == nil {
		t.Error("redis_address flag not registered")
	}
	if flags.Lookup("domain_name_extension") == nil {
		t.Error("domain_name_extension flag not registered")
	}
}

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}
