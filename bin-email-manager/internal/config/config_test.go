package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-email-manager/internal/configkeys"
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

// The provider credentials are read from the environment by emailconfig, not through Config: Config
// has no field for them.
func Test_Config_noProviderSecretFields(t *testing.T) {
	removed := []string{"SendgridAPIKey", "MailgunAPIKey", "MailgunWebhookSigningKey"}

	typ := reflect.TypeOf(Config{})
	for _, name := range removed {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("Wrong match. Config must not have the field %s", name)
		}
	}
	if _, ok := reflect.TypeOf(&Config{}).MethodByName("EnvValue"); ok {
		t.Errorf("Wrong match. Config must not have the EnvValue method")
	}
}

// The provider variables are the source of the credentials, but not through Config: the bootstrap
// binds no flag and no field for them, leaves the other fields alone, and the variables stay
// visible to envsecret.
func Test_Bootstrap_providerVariablesAreNotBoundToConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Setenv("SENDGRID_API_KEY", "synthetic-sendgrid")
	t.Setenv("MAILGUN_API_KEY", "synthetic-mailgun")
	t.Setenv("MAILGUN_WEBHOOK_SIGNING_KEY", "synthetic-signing")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://test-rabbitmq:5672")
	t.Setenv("REDIS_DATABASE", "3")
	t.Setenv("EMAIL_CONFIG_ENCRYPTION_KEYS", "1:synthetic-master")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	for env, want := range map[string]string{
		configkeys.EnvSendgridAPIKey:           "synthetic-sendgrid",
		configkeys.EnvMailgunAPIKey:            "synthetic-mailgun",
		configkeys.EnvMailgunWebhookSigningKey: "synthetic-signing",
	} {
		if got, state := envsecret.Lookup(env); got != want || state != envsecret.Set {
			t.Errorf("Wrong match. %s expect: the environment value to be visible, got state: %s", env, state)
		}
	}

	cfg := Get()
	if cfg.RabbitMQAddress != "amqp://test-rabbitmq:5672" || cfg.RedisDatabase != 3 || cfg.ConfigEncryptionKeys != "1:synthetic-master" {
		t.Errorf("Wrong match. the other fields were affected: %+v", cfg)
	}
	if cfg.DatabaseDSN != "" || cfg.RedisAddress != "" || cfg.RedisPassword != "" {
		t.Errorf("Wrong match. unset fields must stay empty: %+v", cfg)
	}

	for _, name := range []string{"sendgrid_api_key", "mailgun_api_key", "mailgun_webhook_signing_key"} {
		if cmd.PersistentFlags().Lookup(name) != nil || cmd.Flags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q still exists", name)
		}
	}
}
