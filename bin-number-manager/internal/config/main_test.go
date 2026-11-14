package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	config := Get()
	if config == nil {
		t.Error("Expected non-nil config")
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// Reset viper for testing
	viper.Reset()

	// Set some test values
	viper.Set("rabbitmq_address", "amqp://test")
	viper.Set("prometheus_endpoint", "/metrics")
	viper.Set("prometheus_listen_address", ":8080")
	viper.Set("database_dsn", "test_dsn")
	viper.Set("redis_address", "localhost:6379")
	viper.Set("redis_password", "test_pass")
	viper.Set("redis_database", 1)

	// Reset the once to allow re-execution
	once = *new(sync.Once)

	LoadGlobalConfig()

	config := Get()
	if config.RabbitMQAddress != "amqp://test" {
		t.Errorf("Expected RabbitMQAddress to be 'amqp://test', got '%s'", config.RabbitMQAddress)
	}
	if config.PrometheusEndpoint != "/metrics" {
		t.Errorf("Expected PrometheusEndpoint to be '/metrics', got '%s'", config.PrometheusEndpoint)
	}
	if config.PrometheusListenAddress != ":8080" {
		t.Errorf("Expected PrometheusListenAddress to be ':8080', got '%s'", config.PrometheusListenAddress)
	}
	if config.DatabaseDSN != "test_dsn" {
		t.Errorf("Expected DatabaseDSN to be 'test_dsn', got '%s'", config.DatabaseDSN)
	}
	if config.RedisAddress != "localhost:6379" {
		t.Errorf("Expected RedisAddress to be 'localhost:6379', got '%s'", config.RedisAddress)
	}
	if config.RedisPassword != "test_pass" {
		t.Errorf("Expected RedisPassword to be 'test_pass', got '%s'", config.RedisPassword)
	}
	if config.RedisDatabase != 1 {
		t.Errorf("Expected RedisDatabase to be 1, got %d", config.RedisDatabase)
	}
}

func TestBootstrap(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}

	err := Bootstrap(cmd)
	if err != nil {
		t.Errorf("Expected no error from Bootstrap, got %v", err)
	}

	// Verify flags were created
	flag := cmd.PersistentFlags().Lookup("rabbitmq_address")
	if flag == nil {
		t.Error("Expected rabbitmq_address flag to be defined")
	}

	flag = cmd.PersistentFlags().Lookup("database_dsn")
	if flag == nil {
		t.Error("Expected database_dsn flag to be defined")
	}
}

func TestBindConfig(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}

	err := bindConfig(cmd)
	if err != nil {
		t.Errorf("Expected no error from bindConfig, got %v", err)
	}

	// Verify all expected flags are created
	expectedFlags := []string{
		"rabbitmq_address",
		"prometheus_endpoint",
		"prometheus_listen_address",
		"database_dsn",
		"redis_address",
		"redis_password",
		"redis_database",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' to be defined", flagName)
		}
	}
}

// Test_bindConfig_noProviderCredential checks that no provider credential can be set through the
// config any more: the credentials of the providers are read from the database only.
func Test_bindConfig_noProviderCredential(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("Expected no error from bindConfig, got %v", err)
	}

	for _, flagName := range []string{"twilio_sid", "twilio_token", "telnyx_connection_id", "telnyx_profile_id", "telnyx_token"} {
		if cmd.PersistentFlags().Lookup(flagName) != nil {
			t.Errorf("Expected flag '%s' to be removed", flagName)
		}
	}
}
