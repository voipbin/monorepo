package config

import (
	"net/http"
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds process-wide configuration values loaded from command-line
// flags and environment variables for the service.
type Config struct {
	RabbitMQAddress         string // RabbitMQAddress is the address (including host and port) of the RabbitMQ server.
	PrometheusEndpoint      string // PrometheusEndpoint is the HTTP path at which Prometheus metrics are exposed.
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":8080").
	DatabaseDSN             string // DatabaseDSN is the data source name used to connect to the primary database.
	RedisAddress            string // RedisAddress is the address (including host and port) of the Redis server.
	RedisPassword           string // RedisPassword is the password used for authenticating to the Redis server.
	RedisDatabase           int    // RedisDatabase is the numeric Redis logical database index to select, not a name.
	EngineKeyChatGPT        string // EngineKeyChatGPT is the API key for ChatGPT engine.
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}

	return nil
}

// bindConfig binds CLI flags and environment variables for configuration.
// It maps command-line flags to environment variables using Viper.
func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")
	f.String("engine_key_chatgpt", "", "Engine key for chatgpt")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"engine_key_chatgpt":        "ENGINE_KEY_CHATGPT",
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

func Get() *Config {
	return &globalConfig
}

// LoadGlobalConfig loads configuration from viper into the global singleton.
// NOTE: This must be called AFTER Bootstrap (which calls bindConfig) has been executed.
// If called before binding, it will load empty/default values.
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
			EngineKeyChatGPT:        viper.GetString("engine_key_chatgpt"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// InitPrometheus initializes Prometheus metrics server.
// Must be called AFTER LoadGlobalConfig().
func InitPrometheus() {
	cfg := Get()

	// Skip Prometheus initialization if endpoint or listen address is not configured
	if cfg.PrometheusEndpoint == "" || cfg.PrometheusListenAddress == "" {
		logrus.Debug("Prometheus metrics server disabled (endpoint or listen address not configured)")
		return
	}

	http.Handle(cfg.PrometheusEndpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", cfg.PrometheusListenAddress, cfg.PrometheusEndpoint)
		if err := http.ListenAndServe(cfg.PrometheusListenAddress, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}
