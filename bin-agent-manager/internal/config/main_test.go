package config

import (
	"strings"
	"sync"
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

// Test_LoadGlobalConfig verifies that every viper key is copied into its own
// Config field and that the load happens only once. The sync.Once and the
// global config are reset before and after the test so that repeated runs
// start from the same state; viper.Set takes precedence over flags and the
// environment, so the result does not depend on the process environment.
func Test_LoadGlobalConfig(t *testing.T) {
	reset := func() {
		once = sync.Once{}
		globalConfig = Config{}
		viper.Reset()
	}
	reset()
	t.Cleanup(reset)

	// a synthetic value that is built at run time, not a literal credential
	synthetic := strings.Join([]string{"synthetic", "value"}, "-")

	viper.Set("rabbitmq_address", "amqp://mq.invalid:5672")
	viper.Set("prometheus_endpoint", "/test-metrics")
	viper.Set("prometheus_listen_address", ":18080")
	viper.Set("database_dsn", "db.invalid:3306/test")
	viper.Set("redis_address", "203.0.113.10:6379")
	viper.Set("redis_password", synthetic)
	viper.Set("redis_database", 7)
	viper.Set("password_reset_base_url", "https://reset.invalid")

	LoadGlobalConfig()

	got := *Get()
	checks := []struct {
		name     string
		got      any
		expected any
	}{
		{"RedisPassword", got.RedisPassword, synthetic},
		{"RabbitMQAddress", got.RabbitMQAddress, "amqp://mq.invalid:5672"},
		{"PrometheusEndpoint", got.PrometheusEndpoint, "/test-metrics"},
		{"PrometheusListenAddress", got.PrometheusListenAddress, ":18080"},
		{"DatabaseDSN", got.DatabaseDSN, "db.invalid:3306/test"},
		{"RedisDatabase", got.RedisDatabase, 7},
		{"PasswordResetBaseURL", got.PasswordResetBaseURL, "https://reset.invalid"},
		{"RedisAddress", got.RedisAddress, "203.0.113.10:6379"},
	}
	for _, c := range checks {
		if c.got != c.expected {
			t.Errorf("Config.%s = %v, expected %v", c.name, c.got, c.expected)
		}
	}

	// a second call must keep the first values even if viper changed meanwhile
	viper.Set("rabbitmq_address", "amqp://other.invalid:5672")
	viper.Set("redis_database", 9)
	LoadGlobalConfig()
	if again := *Get(); again != got {
		t.Errorf("config changed on the second load.\nexpected: %+v\ngot: %+v", got, again)
	}
}
