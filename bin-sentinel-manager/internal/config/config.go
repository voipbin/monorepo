package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the sentinel-manager service
type Config struct {
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
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
	if err = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint")); err != nil {
		return fmt.Errorf("error binding prometheus_endpoint flag: %w", err)
	}
	if err = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address")); err != nil {
		return fmt.Errorf("error binding prometheus_listen_address flag: %w", err)
	}
	if err = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address")); err != nil {
		return fmt.Errorf("error binding rabbitmq_address flag: %w", err)
	}

	// Load configuration from viper into struct
	cfg = Config{
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),
	}

	return nil
}
