package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	amai "monorepo/bin-ai-manager/models/ai"
)

// Config holds the application configuration
type Config struct {
	DatabaseDSN             string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string
	RedisAddress            string
	RedisDatabase           int
	RedisPassword           string
}

var (
	appConfig Config
	once      sync.Once
)

// Bootstrap initializes configuration with Cobra command and Viper
func Bootstrap(rootCmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(rootCmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}

	return nil
}

// bindConfig binds CLI flags and environment variables for configuration.
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
	f.String("local_llm_models", "", "JSON list of operator-registered local (self-hosted OpenAI-compatible) models; no URL or key")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"local_llm_models":          "LOCAL_LLM_MODELS",
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

// Get returns the application configuration
func Get() Config {
	return appConfig
}

// LoadGlobalConfig loads configuration from viper into the global singleton.
// NOTE: This must be called AFTER Bootstrap has been executed.
// If called before binding, it will load empty/default values.
func LoadGlobalConfig() {
	once.Do(func() {
		appConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// LocalLLMModels returns the LOCAL_LLM_MODELS value. It is read from viper and is not a
// Config field because the daemon logs the whole Config struct at start.
func LocalLLMModels() string {
	return viper.GetString("local_llm_models")
}

// ConfigureLocalModels registers the local models from LOCAL_LLM_MODELS. Call it right
// after LoadGlobalConfig and before anything serves. A bad value never stops the
// service: it logs an error (counts only, never the value) and local models stay empty.
func ConfigureLocalModels() {
	res, err := amai.ConfigureLocalModels(LocalLLMModels())
	log := logrus.WithFields(res.LogFields())
	switch {
	case err != nil:
		log.WithError(err).Error("Could not load local llm models. No local model is available.")
	case res.Skipped > 0 || res.DuplicateDefaults > 0:
		log.Error("Some local llm model entries were skipped.")
	default:
		log.Info("Local llm models loaded.")
	}
}
