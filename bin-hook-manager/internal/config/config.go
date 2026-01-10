package config

import (
	"sync"

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

// InitConfig initializes configuration with Cobra command
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
		panic("config not initialized - call InitConfig first")
	}
	return cfg
}
