package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the tag-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
}

var cfg Config

// Get returns the current configuration
func Get() Config {
	return cfg
}

// InitConfig initializes the configuration with Cobra command
func InitConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	var err error

	// Bind flags to viper
	if err = viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn")); err != nil {
		return fmt.Errorf("error binding database_dsn flag: %w", err)
	}
	if err = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint")); err != nil {
		return fmt.Errorf("error binding prometheus_endpoint flag: %w", err)
	}
	if err = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address")); err != nil {
		return fmt.Errorf("error binding prometheus_listen_address flag: %w", err)
	}
	if err = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address")); err != nil {
		return fmt.Errorf("error binding rabbitmq_address flag: %w", err)
	}
	if err = viper.BindPFlag("redis_address", cmd.Flags().Lookup("redis_address")); err != nil {
		return fmt.Errorf("error binding redis_address flag: %w", err)
	}
	if err = viper.BindPFlag("redis_database", cmd.Flags().Lookup("redis_database")); err != nil {
		return fmt.Errorf("error binding redis_database flag: %w", err)
	}
	if err = viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password")); err != nil {
		return fmt.Errorf("error binding redis_password flag: %w", err)
	}

	// Load configuration from viper into struct
	cfg = Config{
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
