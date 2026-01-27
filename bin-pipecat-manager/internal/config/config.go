package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

var (
	appConfig Config
	once      sync.Once
)

// Bootstrap initializes configuration with Cobra command and Viper
func Bootstrap(rootCmd *cobra.Command) error {
	// Set up logging format
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Enable automatic environment variable reading
	viper.AutomaticEnv()

	// Define flags
	rootCmd.PersistentFlags().String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection")
	rootCmd.PersistentFlags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.PersistentFlags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on")
	rootCmd.PersistentFlags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server")
	rootCmd.PersistentFlags().String("redis_address", "127.0.0.1:6379", "Address of the Redis server")
	rootCmd.PersistentFlags().String("redis_password", "", "Password for authenticating with the Redis server")
	rootCmd.PersistentFlags().Int("redis_database", 1, "Redis database index to use")

	// Bind flags to viper
	if err := viper.BindPFlag("database_dsn", rootCmd.PersistentFlags().Lookup("database_dsn")); err != nil {
		return err
	}
	if err := viper.BindPFlag("prometheus_endpoint", rootCmd.PersistentFlags().Lookup("prometheus_endpoint")); err != nil {
		return err
	}
	if err := viper.BindPFlag("prometheus_listen_address", rootCmd.PersistentFlags().Lookup("prometheus_listen_address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("rabbitmq_address", rootCmd.PersistentFlags().Lookup("rabbitmq_address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("redis_address", rootCmd.PersistentFlags().Lookup("redis_address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("redis_password", rootCmd.PersistentFlags().Lookup("redis_password")); err != nil {
		return err
	}
	if err := viper.BindPFlag("redis_database", rootCmd.PersistentFlags().Lookup("redis_database")); err != nil {
		return err
	}

	// Bind environment variables
	if err := viper.BindEnv("database_dsn", "DATABASE_DSN"); err != nil {
		return err
	}
	if err := viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT"); err != nil {
		return err
	}
	if err := viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS"); err != nil {
		return err
	}
	if err := viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS"); err != nil {
		return err
	}
	if err := viper.BindEnv("redis_address", "REDIS_ADDRESS"); err != nil {
		return err
	}
	if err := viper.BindEnv("redis_password", "REDIS_PASSWORD"); err != nil {
		return err
	}
	if err := viper.BindEnv("redis_database", "REDIS_DATABASE"); err != nil {
		return err
	}

	// Load configuration into struct
	appConfig = Config{
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

// Get returns the application configuration
func Get() Config {
	return appConfig
}

// LoadGlobalConfig loads configuration from viper into the global singleton.
// NOTE: This must be called AFTER Bootstrap has been executed.
// If called before binding, it will load empty/default values.
func LoadGlobalConfig() {
	once.Do(func() {
		appConfig = Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisDatabase:           viper.GetInt("redis_database"),
			RedisPassword:           viper.GetString("redis_password"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}
