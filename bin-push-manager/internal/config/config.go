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

// Config holds all configuration for the push-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	FCMServiceAccountFile   string // path of the FCM service account JSON. empty means FCM is not configured
}

// Get returns the current configuration
func Get() *Config {
	return &globalConfig
}

// bindings maps the flag names to the environment variable names.
var bindings = map[string]string{
	"database_dsn":              "DATABASE_DSN",
	"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
	"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
	"rabbitmq_address":          "RABBITMQ_ADDRESS",
	"fcm_service_account_file":  "FCM_SERVICE_ACCOUNT_FILE",
}

func defineFlags(f interface {
	String(name string, value string, usage string) *string
}) {
	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Database connection DSN")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("fcm_service_account_file", "", "Path of the FCM service account JSON file. Empty means FCM is not configured")
}

// Bootstrap binds CLI flags and environment variables for configuration.
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()
	defineFlags(f)

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
	once.Do(load)
}

func load() {
	globalConfig = Config{
		DatabaseDSN:             viper.GetString("database_dsn"),
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),
		FCMServiceAccountFile:   viper.GetString("fcm_service_account_file"),
	}
}

// InitConfig initializes the configuration with Cobra command (for daemon)
func InitConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	for flagKey, envKey := range bindings {
		if err := viper.BindPFlag(flagKey, cmd.Flags().Lookup(flagKey)); err != nil {
			return errors.Wrapf(err, "error binding %s flag", flagKey)
		}

		if err := viper.BindEnv(flagKey, envKey); err != nil {
			return errors.Wrapf(err, "could not bind the env. key: %s", envKey)
		}
	}

	load()

	return nil
}
