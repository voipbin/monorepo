package config

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg  Config
	once sync.Once
)

// Config holds all configuration for the sentinel-manager service
type Config struct {
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
}

// Get returns the current configuration
func Get() Config {
	return cfg
}

// Bootstrap sets up configuration flags and bindings for CLI tools
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
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

// LoadGlobalConfig loads configuration from viper into the global singleton
func LoadGlobalConfig() {
	once.Do(func() {
		cfg = Config{
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
		}
	})
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
