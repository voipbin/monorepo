package config

import (
	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GlobalConfig holds all validated configurations and is initialized once by InitAll.
// It must be treated as read-only after initialization and must not be modified at runtime.
var GlobalConfig Config

type Config struct {
	RabbitMQAddress      string
	PrometheusEndpoint   string
	PrometheusListenAddr string
	DatabaseDSN          string
	RedisAddress         string
	RedisPassword        string
	RedisDatabase        int
}

func BindConfig(cmd *cobra.Command) error {
	initLog()

	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
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

func LoadGlobalConfig() {
	GlobalConfig = Config{
		RabbitMQAddress:      viper.GetString("rabbitmq_address"),
		PrometheusEndpoint:   viper.GetString("prometheus_endpoint"),
		PrometheusListenAddr: viper.GetString("prometheus_listen_address"),
		DatabaseDSN:          viper.GetString("database_dsn"),
		RedisAddress:         viper.GetString("redis_address"),
		RedisPassword:        viper.GetString("redis_password"),
		RedisDatabase:        viper.GetInt("redis_database"),
	}
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
