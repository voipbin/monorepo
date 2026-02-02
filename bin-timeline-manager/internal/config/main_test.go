package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{}

	// Test that a new Config has empty strings by default
	if cfg.RabbitMQAddress != "" {
		t.Errorf("RabbitMQAddress = %q, want empty", cfg.RabbitMQAddress)
	}

	if cfg.PrometheusEndpoint != "" {
		t.Errorf("PrometheusEndpoint = %q, want empty", cfg.PrometheusEndpoint)
	}

	if cfg.PrometheusListenAddress != "" {
		t.Errorf("PrometheusListenAddress = %q, want empty", cfg.PrometheusListenAddress)
	}

	if cfg.ClickHouseAddress != "" {
		t.Errorf("ClickHouseAddress = %q, want empty", cfg.ClickHouseAddress)
	}

	if cfg.ClickHouseDatabase != "" {
		t.Errorf("ClickHouseDatabase = %q, want empty", cfg.ClickHouseDatabase)
	}

	if cfg.MigrationsPath != "" {
		t.Errorf("MigrationsPath = %q, want empty", cfg.MigrationsPath)
	}
}

func TestConfig_AllFields(t *testing.T) {
	cfg := &Config{
		RabbitMQAddress:         "amqp://localhost:5672",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		ClickHouseAddress:       "localhost:9000",
		ClickHouseDatabase:      "voipbin",
		MigrationsPath:          "/app/migrations",
	}

	if cfg.RabbitMQAddress != "amqp://localhost:5672" {
		t.Errorf("RabbitMQAddress = %q, want %q", cfg.RabbitMQAddress, "amqp://localhost:5672")
	}

	if cfg.PrometheusEndpoint != "/metrics" {
		t.Errorf("PrometheusEndpoint = %q, want %q", cfg.PrometheusEndpoint, "/metrics")
	}

	if cfg.PrometheusListenAddress != ":2112" {
		t.Errorf("PrometheusListenAddress = %q, want %q", cfg.PrometheusListenAddress, ":2112")
	}

	if cfg.ClickHouseAddress != "localhost:9000" {
		t.Errorf("ClickHouseAddress = %q, want %q", cfg.ClickHouseAddress, "localhost:9000")
	}

	if cfg.ClickHouseDatabase != "voipbin" {
		t.Errorf("ClickHouseDatabase = %q, want %q", cfg.ClickHouseDatabase, "voipbin")
	}

	if cfg.MigrationsPath != "/app/migrations" {
		t.Errorf("MigrationsPath = %q, want %q", cfg.MigrationsPath, "/app/migrations")
	}
}

func TestBindConfig(t *testing.T) {
	cmd := &cobra.Command{}
	err := bindConfig(cmd)
	if err != nil {
		t.Fatalf("bindConfig() error = %v", err)
	}

	// Verify flags were registered
	flags := []string{
		"rabbitmq_address",
		"prometheus_endpoint",
		"prometheus_listen_address",
		"clickhouse_address",
		"clickhouse_database",
		"migrations_path",
	}

	for _, flagName := range flags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %q was not registered", flagName)
		}
	}
}

func TestBootstrap(t *testing.T) {
	cmd := &cobra.Command{}
	err := Bootstrap(cmd)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
}

func TestInitLog(t *testing.T) {
	// initLog should not panic
	initLog()
}

func TestLoadGlobalConfig(t *testing.T) {
	// LoadGlobalConfig uses sync.Once, so it will only run once per test run.
	// We test that it doesn't panic.
	LoadGlobalConfig()

	// After loading, Get() should return the same instance
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil after LoadGlobalConfig()")
	}
}

func TestLoadGlobalConfig_IdempotentCall(t *testing.T) {
	// Multiple calls to LoadGlobalConfig should be idempotent
	LoadGlobalConfig()
	LoadGlobalConfig()
	LoadGlobalConfig()

	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil after multiple LoadGlobalConfig() calls")
	}
}
