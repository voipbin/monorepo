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
// flags and environment variables for the service.
type Config struct {
	RabbitMQAddress         string // RabbitMQAddress is the address (including host and port) of the RabbitMQ server.
	PrometheusEndpoint      string // PrometheusEndpoint is the HTTP path at which Prometheus metrics are exposed.
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":2112").
	DatabaseDSN             string // DatabaseDSN is the data source name used to connect to the primary database.
	RedisAddress            string // RedisAddress is the address (including host and port) of the Redis server.
	RedisPassword           string // RedisPassword is the password used for authenticating to the Redis server.
	RedisDatabase           int    // RedisDatabase is the numeric Redis logical database index to select, not a name.

	ScheduleTickIntervalSec        int    // ScheduleTickIntervalSec is the dispatch loop scan cadence in seconds.
	ScheduleDispatchConcurrency    int    // ScheduleDispatchConcurrency is the max number of in-flight dispatches per replica.
	ScheduleExecutionRetentionDays int    // ScheduleExecutionRetentionDays is the age in days after which execution rows are pruned.
	ScheduleBackupDir              string // ScheduleBackupDir is the directory where database backup dumps are written. No default; must be set explicitly to enable backups.
	ScheduleBackupRetentionCount   int    // ScheduleBackupRetentionCount is the number of newest backup files to keep.
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
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 1, "Redis database index")
	f.Int("schedule_tick_interval_sec", 10, "Dispatch loop scan cadence in seconds")
	f.Int("schedule_dispatch_concurrency", 10, "Max number of in-flight dispatches per replica")
	f.Int("schedule_execution_retention_days", 90, "Age in days after which execution rows are pruned")
	f.String("schedule_backup_dir", "", "Directory where database backup dumps are written")
	f.Int("schedule_backup_retention_count", 7, "Number of newest backup files to keep")

	bindings := map[string]string{
		"rabbitmq_address":                   "RABBITMQ_ADDRESS",
		"prometheus_endpoint":                "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address":          "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":                       "DATABASE_DSN",
		"redis_address":                      "REDIS_ADDRESS",
		"redis_password":                     "REDIS_PASSWORD",
		"redis_database":                     "REDIS_DATABASE",
		"schedule_tick_interval_sec":        "SCHEDULE_TICK_INTERVAL_SEC",
		"schedule_dispatch_concurrency":     "SCHEDULE_DISPATCH_CONCURRENCY",
		"schedule_execution_retention_days": "SCHEDULE_EXECUTION_RETENTION_DAYS",
		"schedule_backup_dir":               "SCHEDULE_BACKUP_DIR",
		"schedule_backup_retention_count":   "SCHEDULE_BACKUP_RETENTION_COUNT",
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

			ScheduleTickIntervalSec:        viper.GetInt("schedule_tick_interval_sec"),
			ScheduleDispatchConcurrency:    viper.GetInt("schedule_dispatch_concurrency"),
			ScheduleExecutionRetentionDays: viper.GetInt("schedule_execution_retention_days"),
			ScheduleBackupDir:              viper.GetString("schedule_backup_dir"),
			ScheduleBackupRetentionCount:   viper.GetInt("schedule_backup_retention_count"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
