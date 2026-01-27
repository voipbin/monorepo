package config

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the hook-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	SSLPrivkeyBase64        string
	SSLCertBase64           string
}

var (
	cfg  *Config
	once sync.Once
)

// Bootstrap binds configuration flags to the root command
func Bootstrap(cmd *cobra.Command) error {
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}
	return nil
}

// bindConfig binds CLI flags and environment variables for configuration.
func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Database connection DSN")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("ssl_privkey_base64", "", "Base64-encoded SSL private key")
	f.String("ssl_cert_base64", "", "Base64-encoded SSL certificate")

	bindings := map[string]string{
		"database_dsn":              "DATABASE_DSN",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"ssl_privkey_base64":        "SSL_PRIVKEY_BASE64",
		"ssl_cert_base64":           "SSL_CERT_BASE64",
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
// NOTE: This must be called AFTER Bootstrap has been executed.
func LoadGlobalConfig() {
	once.Do(func() {
		cfg = &Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			SSLPrivkeyBase64:        viper.GetString("ssl_privkey_base64"),
			SSLCertBase64:           viper.GetString("ssl_cert_base64"),
		}
	})
}

// InitConfig initializes configuration with Cobra command (legacy function for hook-manager main)
func InitConfig(cmd *cobra.Command) {
	once.Do(func() {
		viper.AutomaticEnv()

		// Bind flags to viper
		_ = viper.BindPFlag("database_dsn", cmd.Flags().Lookup("database_dsn"))
		_ = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint"))
		_ = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address"))
		_ = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address"))
		_ = viper.BindPFlag("ssl_privkey_base64", cmd.Flags().Lookup("ssl_privkey_base64"))
		_ = viper.BindPFlag("ssl_cert_base64", cmd.Flags().Lookup("ssl_cert_base64"))

		cfg = &Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			SSLPrivkeyBase64:        viper.GetString("ssl_privkey_base64"),
			SSLCertBase64:           viper.GetString("ssl_cert_base64"),
		}
	})
}

// Get returns the global config instance
func Get() *Config {
	if cfg == nil {
		panic("config not initialized - call InitConfig or LoadGlobalConfig first")
	}
	return cfg
}
