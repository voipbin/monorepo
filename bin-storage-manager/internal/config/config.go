package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds all configuration for the storage-manager service
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
	GCPProjectID            string
	GCPBucketNameMedia      string
	GCPBucketNameTmp        string
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

	f.String("database_dsn", "", "Database connection DSN")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")
	f.String("gcp_project_id", "", "GCP project ID")
	f.String("gcp_bucket_name_media", "", "GCP bucket name for media storage")
	f.String("gcp_bucket_name_tmp", "", "GCP bucket name for temporary storage")

	bindings := map[string]string{
		"database_dsn":              "DATABASE_DSN",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"gcp_project_id":            "GCP_PROJECT_ID",
		"gcp_bucket_name_media":     "GCP_BUCKET_NAME_MEDIA",
		"gcp_bucket_name_tmp":       "GCP_BUCKET_NAME_TMP",
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
			DatabaseDSN:             viper.GetString("database_dsn"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisDatabase:           viper.GetInt("redis_database"),
			RedisPassword:           viper.GetString("redis_password"),
			GCPProjectID:            viper.GetString("gcp_project_id"),
			GCPBucketNameMedia:      viper.GetString("gcp_bucket_name_media"),
			GCPBucketNameTmp:        viper.GetString("gcp_bucket_name_tmp"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
