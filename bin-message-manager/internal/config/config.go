package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Names of the environment variables that the service reads without a flag.
const (
	// EnvConfigEncryptionKeys is the environment variable of the master key list of the stored
	// settings.
	EnvConfigEncryptionKeys = "MESSAGE_CONFIG_ENCRYPTION_KEYS"
)

// Config holds all configuration for the message-manager service
type Config struct {
	DatabaseDSN             string // DatabaseDSN is the data source name used to connect to the primary database.
	PrometheusEndpoint      string // PrometheusEndpoint is the HTTP path at which Prometheus metrics are exposed.
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":8080").
	RabbitMQAddress         string // RabbitMQAddress is the address (including host and port) of the RabbitMQ server.
	RedisAddress            string // RedisAddress is the address (including host and port) of the Redis server.
	RedisDatabase           int    // RedisDatabase is the numeric Redis logical database index to select, not a name.
	RedisPassword           string // RedisPassword is the password used for authenticating to the Redis server.

	// ConfigEncryptionKeys is the master key list of the stored settings
	// ("<version>:<base64 32 bytes>[,...]"). It is read from the environment only, never from a
	// command line flag. A value that can not be parsed does not stop the service: the stored
	// secrets can not be used then (they are reported as undecryptable).
	ConfigEncryptionKeys string
}

var (
	globalConfig Config
	once         sync.Once
)

// Bootstrap binds CLI flags and environment variables for configuration.
// It maps command-line flags to environment variables using Viper.
func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}

	return nil
}

// bindConfig binds CLI flags and environment variables for configuration.
// It maps command-line flags to environment variables using Viper.
func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
	}

	// the master key list of the stored settings has no flag: a flag value would show in the
	// process list.
	if errBind := viper.BindEnv("message_config_encryption_keys", EnvConfigEncryptionKeys); errBind != nil {
		return errors.Wrapf(errBind, "could not bind the env. key: %s", EnvConfigEncryptionKeys)
	}

	for flagKey, envKey := range bindings {
		if errBind := viper.BindPFlag(flagKey, f.Lookup(flagKey)); errBind != nil {
			return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
		}

		if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
			return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
		}
	}

	return nil
}

// Get returns the global config instance
func Get() *Config {
	return &globalConfig
}

// LoadGlobalConfig loads configuration from viper into the global singleton.
// NOTE: This must be called AFTER Bootstrap (which calls bindConfig) has been executed.
// If called before binding, it will load empty/default values.
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
			ConfigEncryptionKeys:    viper.GetString("message_config_encryption_keys"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// RegisterFlags registers command-line flags for the service configuration.
// This is a backward-compatible wrapper that delegates to bindConfig.
func RegisterFlags(cmd *cobra.Command) {
	initLog()
	if err := bindConfig(cmd); err != nil {
		logrus.Fatalf("Failed to register flags: %v", err)
	}
}

// InitConfig initializes the configuration by loading values from flags and environment variables.
// This is a backward-compatible wrapper that delegates to LoadGlobalConfig.
func InitConfig(cmd *cobra.Command) {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		logrus.Fatalf("Failed to bind flags: %v", err)
	}
	LoadGlobalConfig()
}
