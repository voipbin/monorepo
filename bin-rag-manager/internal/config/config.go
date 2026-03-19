package config

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds all configuration for the rag-manager service
type Config struct {
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string

	// Google Cloud / Vertex AI
	GoogleCloudProject   string
	GoogleCloudLocation  string
	GoogleEmbeddingModel string

	// RAG
	RAGTopK int

	// GCS
	GCPBucketNameMedia string

	// PostgreSQL
	PostgreSQLDSN string
}

// Get returns the current configuration
func Get() *Config {
	return &globalConfig
}

// Bootstrap binds CLI flags and environment variables for configuration.
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("gcp_project_id", "", "GCP project ID for Vertex AI")
	f.String("gcp_location", "", "GCP region for Vertex AI")
	f.String("google_embedding_model", "text-embedding-004", "Google embedding model")
	f.Int("rag_top_k", 5, "Number of chunks to retrieve")
	f.String("gcp_bucket_name_media", "", "GCS bucket name for media files")
	f.String("postgresql_dsn", "", "PostgreSQL connection string")

	bindings := map[string]string{
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"gcp_project_id":            "GCP_PROJECT_ID",
		"gcp_location":              "GCP_LOCATION",
		"google_embedding_model":    "GOOGLE_EMBEDDING_MODEL",
		"rag_top_k":                 "RAG_TOP_K",
		"gcp_bucket_name_media":     "GCP_BUCKET_NAME_MEDIA",
		"postgresql_dsn":            "POSTGRESQL_DSN",
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

// LoadGlobalConfig loads configuration from viper into the global singleton.
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			GoogleCloudProject:      viper.GetString("gcp_project_id"),
			GoogleCloudLocation:     viper.GetString("gcp_location"),
			GoogleEmbeddingModel:    viper.GetString("google_embedding_model"),
			RAGTopK:                 viper.GetInt("rag_top_k"),
			GCPBucketNameMedia:      viper.GetString("gcp_bucket_name_media"),
			PostgreSQLDSN:           viper.GetString("postgresql_dsn"),
		}
	})
}

// InitConfig initializes the configuration with Cobra command (for daemon)
func InitConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	var err error

	if err = viper.BindPFlag("prometheus_endpoint", cmd.Flags().Lookup("prometheus_endpoint")); err != nil {
		return errors.Wrapf(err, "error binding prometheus_endpoint flag")
	}
	if err = viper.BindPFlag("prometheus_listen_address", cmd.Flags().Lookup("prometheus_listen_address")); err != nil {
		return errors.Wrapf(err, "error binding prometheus_listen_address flag")
	}
	if err = viper.BindPFlag("rabbitmq_address", cmd.Flags().Lookup("rabbitmq_address")); err != nil {
		return errors.Wrapf(err, "error binding rabbitmq_address flag")
	}
	if err = viper.BindPFlag("gcp_project_id", cmd.Flags().Lookup("gcp_project_id")); err != nil {
		return errors.Wrapf(err, "error binding gcp_project_id flag")
	}
	if err = viper.BindPFlag("gcp_location", cmd.Flags().Lookup("gcp_location")); err != nil {
		return errors.Wrapf(err, "error binding gcp_location flag")
	}
	if err = viper.BindPFlag("google_embedding_model", cmd.Flags().Lookup("google_embedding_model")); err != nil {
		return errors.Wrapf(err, "error binding google_embedding_model flag")
	}
	if err = viper.BindPFlag("rag_top_k", cmd.Flags().Lookup("rag_top_k")); err != nil {
		return errors.Wrapf(err, "error binding rag_top_k flag")
	}
	if err = viper.BindPFlag("gcp_bucket_name_media", cmd.Flags().Lookup("gcp_bucket_name_media")); err != nil {
		return errors.Wrapf(err, "error binding gcp_bucket_name_media flag")
	}
	if err = viper.BindPFlag("postgresql_dsn", cmd.Flags().Lookup("postgresql_dsn")); err != nil {
		return errors.Wrapf(err, "error binding postgresql_dsn flag")
	}

	globalConfig = Config{
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),
		GoogleCloudProject:      viper.GetString("gcp_project_id"),
		GoogleCloudLocation:     viper.GetString("gcp_location"),
		GoogleEmbeddingModel:    viper.GetString("google_embedding_model"),
		RAGTopK:                 viper.GetInt("rag_top_k"),
		GCPBucketNameMedia:      viper.GetString("gcp_bucket_name_media"),
		PostgreSQLDSN:           viper.GetString("postgresql_dsn"),
	}

	return nil
}
