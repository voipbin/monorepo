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

// Config holds process-wide configuration values.
type Config struct {
	RabbitMQAddress         string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	ClickHouseAddress       string
	ClickHouseDatabase      string
	MigrationsPath          string
	HomerAPIAddress         string
	HomerAuthToken          string
	GCSBucketName           string

	// MySQL (analysis store).
	DatabaseDSN string

	// Analysis stage models (must be in the ai-manager gateway allow-set).
	AnalysisModelStage1 string
	AnalysisModelStage2 string
	AnalysisModelStage3 string
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}
	return nil
}

func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("clickhouse_address", "", "ClickHouse server address")
	f.String("clickhouse_database", "default", "ClickHouse database name")
	f.String("migrations_path", "./migrations", "Path to migration files")
	f.String("homer_api_address", "", "Homer API address")
	f.String("homer_auth_token", "", "Homer auth token")
	f.String("gcs_bucket_name", "", "GCS bucket for RTP pcap recordings")
	f.String("database_dsn", "", "MySQL DSN for the analysis store")
	f.String("analysis_model_stage1", "", "LLM model for analysis stage 1 (inventory)")
	f.String("analysis_model_stage2", "", "LLM model for analysis stage 2 (content)")
	f.String("analysis_model_stage3", "", "LLM model for analysis stage 3 (diagnosis / combined)")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"clickhouse_address":        "CLICKHOUSE_ADDRESS",
		"clickhouse_database":       "CLICKHOUSE_DATABASE",
		"migrations_path":           "MIGRATIONS_PATH",
		"homer_api_address":         "HOMER_API_ADDRESS",
		"homer_auth_token":          "HOMER_AUTH_TOKEN",
		"gcs_bucket_name":           "GCS_BUCKET_NAME",
		"database_dsn":              "DATABASE_DSN",
		"analysis_model_stage1":     "ANALYSIS_MODEL_STAGE1",
		"analysis_model_stage2":     "ANALYSIS_MODEL_STAGE2",
		"analysis_model_stage3":     "ANALYSIS_MODEL_STAGE3",
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

func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			ClickHouseAddress:       viper.GetString("clickhouse_address"),
			ClickHouseDatabase:      viper.GetString("clickhouse_database"),
			MigrationsPath:          viper.GetString("migrations_path"),
			HomerAPIAddress:         viper.GetString("homer_api_address"),
			HomerAuthToken:          viper.GetString("homer_auth_token"),
			GCSBucketName:           viper.GetString("gcs_bucket_name"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			AnalysisModelStage1:     viper.GetString("analysis_model_stage1"),
			AnalysisModelStage2:     viper.GetString("analysis_model_stage2"),
			AnalysisModelStage3:     viper.GetString("analysis_model_stage3"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
