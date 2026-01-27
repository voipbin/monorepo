package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds the application configuration
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
}

// Bootstrap initializes configuration with Cobra command and Viper
func Bootstrap(rootCmd *cobra.Command) error {
	// Set up logging format
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Enable automatic environment variable reading
	viper.AutomaticEnv()
	f := rootCmd.PersistentFlags()

	// Define flags
	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection")
	f.String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on")
	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server")
	f.String("redis_address", "127.0.0.1:6379", "Address of the Redis server")
	f.String("redis_password", "", "Password for authenticating with the Redis server")
	f.Int("redis_database", 1, "Redis database index to use")

	bindings := map[string]string{
		"database_dsn":              "DATABASE_DSN",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
	}

	for flagKey, envKey := range bindings {
		if errBind := viper.BindPFlag(flagKey, f.Lookup(flagKey)); errBind != nil {
			return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
		}

		if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
			return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
		}
	}

	// Load configuration into struct for daemon mode
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

// Get returns the application configuration
func Get() *Config {
	return &globalConfig
}
