package config

import (
	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Default Values
const (
	defaultDatabaseDSN             = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultRedisAddress            = "127.0.0.1:6379"
	defaultRedisDatabase           = 1
	defaultRedisPassword           = ""
)

// GlobalConfig holds all validated configurations
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

func InitAll() error {
	initLog()

	if errVariable := initVariable(); errVariable != nil {
		return errors.Wrap(errVariable, "could not init variable")
	}

	return nil
}

func ParseFlags() {
	pflag.Parse()
}

func initVariable() error {
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "RabbitMQ server address")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Prometheus listen address")
	pflag.String("database_dsn", defaultDatabaseDSN, "Database connection DSN")
	pflag.String("redis_address", defaultRedisAddress, "Redis server address")
	pflag.String("redis_password", defaultRedisPassword, "Redis password")
	pflag.Int("redis_database", defaultRedisDatabase, "Redis database index")

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
		if errBind := viper.BindPFlag(flagKey, pflag.Lookup(flagKey)); errBind != nil {
			return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
		}

		if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
			return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
		}
	}

	GlobalConfig = Config{
		RabbitMQAddress:      viper.GetString("rabbitmq_address"),
		PrometheusEndpoint:   viper.GetString("prometheus_endpoint"),
		PrometheusListenAddr: viper.GetString("prometheus_listen_address"),
		DatabaseDSN:          viper.GetString("database_dsn"),
		RedisAddress:         viper.GetString("redis_address"),
		RedisPassword:        viper.GetString("redis_password"),
		RedisDatabase:        viper.GetInt("redis_database"),
	}

	return nil
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
