package config

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds all configuration for the route-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
	HealthCheckInterval     time.Duration
	SipLBIP                 string
	SipLBPort               int
}

// Get returns the current configuration
func Get() *Config {
	return &globalConfig
}

// Bootstrap binds CLI flags and environment variables for configuration
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Database connection DSN")
	f.String("redis_address", "127.0.0.1:6379", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 1, "Redis database index")
	f.Duration("health_check_interval", 30*time.Second, "Provider health check interval")
	f.String("sip_lb_ip", "", "SIP load balancer IP address registered on Telnyx for IP-auth")
	f.Int("sip_lb_port", 5060, "SIP load balancer port registered on Telnyx")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"health_check_interval":     "HEALTH_CHECK_INTERVAL",
		"sip_lb_ip":                 "SIP_LB_IP",
		"sip_lb_port":               "SIP_LB_PORT",
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
// NOTE: This must be called AFTER Bootstrap has been executed
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisDatabase:           viper.GetInt("redis_database"),
			RedisPassword:           viper.GetString("redis_password"),
			HealthCheckInterval:     viper.GetDuration("health_check_interval"),
			SipLBIP:                 viper.GetString("sip_lb_ip"),
			SipLBPort:               viper.GetInt("sip_lb_port"),
		}
	})
}
