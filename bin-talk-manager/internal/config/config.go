package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	// Database
	DatabaseDSN string

	// RabbitMQ
	RabbitmqAddress   string
	RabbitQueueListen string
	RabbitQueueEvent  string
	RabbitDelayQueue  string

	// Redis
	RedisAddress  string
	RedisPassword string
	RedisDatabase int

	// Prometheus
	PrometheusEndpoint      string
	PrometheusListenAddress string
}

func InitConfig(cmd *cobra.Command) *Config {
	cfg := &Config{}

	// Database
	cmd.Flags().String("database_dsn", "", "Database DSN")
	viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn"))
	viper.SetDefault("database_dsn", "root:password@tcp(localhost:3306)/voipbin")

	// RabbitMQ
	cmd.Flags().String("rabbitmq_address", "", "RabbitMQ address")
	viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address"))
	viper.SetDefault("rabbitmq_address", "amqp://guest:guest@localhost:5672/")

	cmd.Flags().String("rabbit_queue_listen", "", "RabbitMQ listen queue")
	viper.BindPFlag("rabbit_queue_listen", cmd.Flags().Lookup("rabbit_queue_listen"))
	viper.SetDefault("rabbit_queue_listen", "bin-manager.talk.request")

	cmd.Flags().String("rabbit_queue_event", "", "RabbitMQ event queue")
	viper.BindPFlag("rabbit_queue_event", cmd.Flags().Lookup("rabbit_queue_event"))
	viper.SetDefault("rabbit_queue_event", "bin-manager.talk.event")

	cmd.Flags().String("rabbit_delay_queue", "", "RabbitMQ delay queue")
	viper.BindPFlag("rabbit_delay_queue", cmd.Flags().Lookup("rabbit_delay_queue"))
	viper.SetDefault("rabbit_delay_queue", "bin-manager.delay")

	// Redis
	cmd.Flags().String("redis_address", "", "Redis address")
	viper.BindPFlag("redis_address", cmd.Flags().Lookup("redis_address"))
	viper.SetDefault("redis_address", "localhost:6379")

	cmd.Flags().String("redis_password", "", "Redis password")
	viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password"))
	viper.SetDefault("redis_password", "")

	cmd.Flags().Int("redis_database", 0, "Redis database")
	viper.BindPFlag("redis_database", cmd.Flags().Lookup("redis_database"))
	viper.SetDefault("redis_database", 1)

	// Prometheus
	cmd.Flags().String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint"))
	viper.SetDefault("prometheus_endpoint", "/metrics")

	cmd.Flags().String("prometheus_listen_address", "", "Prometheus listen address")
	viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address"))
	viper.SetDefault("prometheus_listen_address", ":2112")

	// Read config
	viper.AutomaticEnv()

	cfg.DatabaseDSN = viper.GetString("database_dsn")
	cfg.RabbitmqAddress = viper.GetString("rabbitmq_address")
	cfg.RabbitQueueListen = viper.GetString("rabbit_queue_listen")
	cfg.RabbitQueueEvent = viper.GetString("rabbit_queue_event")
	cfg.RabbitDelayQueue = viper.GetString("rabbit_delay_queue")
	cfg.RedisAddress = viper.GetString("redis_address")
	cfg.RedisPassword = viper.GetString("redis_password")
	cfg.RedisDatabase = viper.GetInt("redis_database")
	cfg.PrometheusEndpoint = viper.GetString("prometheus_endpoint")
	cfg.PrometheusListenAddress = viper.GetString("prometheus_listen_address")

	return cfg
}
