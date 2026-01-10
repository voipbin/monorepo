package config

import (
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the configuration for the conversation-manager service
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
	cfg  *Config
	once sync.Once
)

const (
	defaultDatabaseDSN             = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultRedisAddress            = "127.0.0.1:6379"
	defaultRedisDatabase           = 1
	defaultRedisPassword           = ""
)

// Init initializes the configuration with Cobra command
func Init(cmd *cobra.Command) {
	once.Do(func() {
		viper.AutomaticEnv()

		// Bind flags to viper
		_ = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address"))
		_ = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint"))
		_ = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address"))
		_ = viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn"))
		_ = viper.BindPFlag("redis_address", cmd.Flags().Lookup("redis_address"))
		_ = viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password"))
		_ = viper.BindPFlag("redis_database", cmd.Flags().Lookup("redis_database"))

		// Bind environment variables
		_ = viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS")
		_ = viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT")
		_ = viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS")
		_ = viper.BindEnv("database_dsn", "DATABASE_DSN")
		_ = viper.BindEnv("redis_address", "REDIS_ADDRESS")
		_ = viper.BindEnv("redis_password", "REDIS_PASSWORD")
		_ = viper.BindEnv("redis_database", "REDIS_DATABASE")

		// Load configuration
		cfg = &Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
		}
	})
}

// Get returns the global configuration instance
func Get() *Config {
	if cfg == nil {
		panic("config not initialized - call Init() first")
	}
	return cfg
}

// RegisterFlags registers all configuration flags with the cobra command
func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	cmd.Flags().String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	cmd.Flags().String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	cmd.Flags().String("database_dsn", defaultDatabaseDSN, "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	cmd.Flags().String("redis_address", defaultRedisAddress, "Address of the Redis server (e.g., localhost:6379)")
	cmd.Flags().String("redis_password", defaultRedisPassword, "Password for authenticating with the Redis server (if required)")
	cmd.Flags().Int("redis_database", defaultRedisDatabase, "Redis database index to use (default is 1)")
}
