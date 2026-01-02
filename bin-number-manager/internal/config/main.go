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

// Config holds process-wide configuration values loaded from command-line
// flags and environment variables for the bin-number-manager service.
type Config struct {
	RabbitMQAddress         string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	DatabaseDSN             string
	RedisAddress            string
	RedisPassword           string
	RedisDatabase           int

	TelnyxConnectionID string
	TelnyxProfileID    string
	TelnyxToken        string

	TwilioSID   string
	TwilioToken string
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
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

	f.String("telnyx_connection_id", "", "Telnyx connection ID")
	f.String("telnyx_profile_id", "", "Telnyx profile ID")
	f.String("telnyx_token", "", "Telnyx API token")

	f.String("twilio_sid", "", "Twilio account SID")
	f.String("twilio_token", "", "Twilio auth token")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"telnyx_connection_id":      "TELNYX_CONNECTION_ID",
		"telnyx_profile_id":         "TELNYX_PROFILE_ID",
		"telnyx_token":              "TELNYX_TOKEN",
		"twilio_sid":                "TWILIO_SID",
		"twilio_token":              "TWILIO_TOKEN",
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

			TelnyxConnectionID: viper.GetString("telnyx_connection_id"),
			TelnyxProfileID:    viper.GetString("telnyx_profile_id"),
			TelnyxToken:        viper.GetString("telnyx_token"),

			TwilioSID:   viper.GetString("twilio_sid"),
			TwilioToken: viper.GetString("twilio_token"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
