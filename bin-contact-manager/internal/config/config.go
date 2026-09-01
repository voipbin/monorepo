package config

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds all configuration for the contact-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
}

// Get returns the current configuration
func Get() *Config {
	return &globalConfig
}

// Bootstrap binds CLI flags and environment variables for configuration.
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Database connection DSN")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("redis_address", "127.0.0.1:6379", "Redis server address")
	f.String("redis_database", "1", "Redis database index")
	f.String("redis_password", "", "Redis password")

	bindings := map[string]string{
		"database_dsn":              "DATABASE_DSN",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"redis_address":             "REDIS_ADDRESS",
		"redis_database":            "REDIS_DATABASE",
		"redis_password":            "REDIS_PASSWORD",
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

// LoadGlobalConfig loads configuration from viper into the global singleton.
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisDatabase:           viper.GetInt("redis_database"),
			RedisPassword:           viper.GetString("redis_password"),
		}
	})
}

// InitConfig initializes the configuration with Cobra command (for daemon)
func InitConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	var err error

	// Bind flags to viper
	if err = viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn")); err != nil {
		return errors.Wrapf(err, "error binding database_dsn flag")
	}
	if err = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint")); err != nil {
		return errors.Wrapf(err, "error binding prometheus_endpoint flag")
	}
	if err = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address")); err != nil {
		return errors.Wrapf(err, "error binding prometheus_listen_address flag")
	}
	if err = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address")); err != nil {
		return errors.Wrapf(err, "error binding rabbitmq_address flag")
	}
	if err = viper.BindPFlag("redis_address", cmd.Flags().Lookup("redis_address")); err != nil {
		return errors.Wrapf(err, "error binding redis_address flag")
	}
	if err = viper.BindPFlag("redis_database", cmd.Flags().Lookup("redis_database")); err != nil {
		return errors.Wrapf(err, "error binding redis_database flag")
	}
	if err = viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password")); err != nil {
		return errors.Wrapf(err, "error binding redis_password flag")
	}

	// Load configuration from viper into struct
	globalConfig = Config{
		DatabaseDSN:             viper.GetString("database_dsn"),
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),
		RedisAddress:            viper.GetString("redis_address"),
		RedisDatabase:           viper.GetInt("redis_database"),
		RedisPassword:           viper.GetString("redis_password"),
	}

	return nil
}
