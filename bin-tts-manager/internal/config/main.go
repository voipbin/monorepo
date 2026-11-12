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

// Names of the environment variables that the service reads without a flag.
const (
	// EnvConfigEncryptionKeys is the environment variable of the master key list of the stored
	// settings.
	EnvConfigEncryptionKeys = "TTS_CONFIG_ENCRYPTION_KEYS"
)

// Config holds process-wide configuration values loaded from command-line
// flags and environment variables for the service.
type Config struct {
	RabbitMQAddress         string // RabbitMQAddress is the address (including host and port) of the RabbitMQ server.
	PrometheusEndpoint      string // PrometheusEndpoint is the HTTP path at which Prometheus metrics are exposed.
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":8080").
	GCPTTSEndpoint          string // GCPTTSEndpoint is the GCP Text-to-Speech API endpoint.
	DatabaseDSN             string // DatabaseDSN is the MySQL database connection string.
	RedisAddress            string // RedisAddress is the Redis server address.
	RedisPassword           string // RedisPassword is the Redis server password.
	RedisDB                 int    // RedisDB is the Redis database number.

	// ConfigEncryptionKeys is the master key list of the stored settings
	// ("<version>:<base64 32 bytes>[,...]"). It is read from the environment only, never from a
	// command line flag. A value that can not be parsed does not stop the service: stored
	// secrets can not be used then (they are reported as undecryptable).
	ConfigEncryptionKeys string
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
	f.String("gcp_tts_endpoint", "", "GCP TTS endpoint")
	f.String("database_dsn", "", "MySQL database connection string")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis server password")
	f.Int("redis_db", 0, "Redis database number")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"gcp_tts_endpoint":          "GCP_TTS_ENDPOINT",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_db":                  "REDIS_DB",
	}

	// the master key list of the stored settings has no flag: a flag value would show in the
	// process list.
	if errBind := viper.BindEnv("tts_config_encryption_keys", EnvConfigEncryptionKeys); errBind != nil {
		return errors.Wrapf(errBind, "could not bind the env. key: %s", EnvConfigEncryptionKeys)
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
			GCPTTSEndpoint:          viper.GetString("gcp_tts_endpoint"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDB:                 viper.GetInt("redis_db"),
			ConfigEncryptionKeys:    viper.GetString("tts_config_encryption_keys"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// ResetForTest clears the global config and lets LoadGlobalConfig load from viper and the
// environment again (LoadGlobalConfig is sync.Once-guarded). A test that needs it must reset
// again in t.Cleanup, so a value it set does not stay in the global config.
// USE ONLY FROM TESTS.
func ResetForTest() {
	once = sync.Once{}
	globalConfig = Config{}
}
