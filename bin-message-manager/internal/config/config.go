package config

import (
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the message-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
	AuthtokenMessagebird    string
	AuthtokenTelnyx         string
}

var (
	cfg  *Config
	once sync.Once
)

// InitConfig initializes configuration with Cobra command
func InitConfig(cmd *cobra.Command) {
	once.Do(func() {
		viper.AutomaticEnv()

		// Bind flags to viper
		_ = viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn"))
		_ = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint"))
		_ = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address"))
		_ = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address"))
		_ = viper.BindPFlag("redis_address", cmd.Flags().Lookup("redis_address"))
		_ = viper.BindPFlag("redis_database", cmd.Flags().Lookup("redis_database"))
		_ = viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password"))
		_ = viper.BindPFlag("authtoken_messagebird", cmd.Flags().Lookup("authtoken_messagebird"))
		_ = viper.BindPFlag("authtoken_telnyx", cmd.Flags().Lookup("authtoken_telnyx"))

		cfg = &Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisDatabase:           viper.GetInt("redis_database"),
			RedisPassword:           viper.GetString("redis_password"),
			AuthtokenMessagebird:    viper.GetString("authtoken_messagebird"),
			AuthtokenTelnyx:         viper.GetString("authtoken_telnyx"),
		}
	})
}

// Get returns the global config instance
func Get() *Config {
	if cfg == nil {
		panic("config not initialized - call InitConfig first")
	}
	return cfg
}
