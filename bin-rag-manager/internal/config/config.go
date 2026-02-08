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

	// OpenAI
	OpenAIAPIKey        string
	OpenAIEmbeddingModel string

	// RAG
	RAGLLMModel       string
	RAGTopK           int
	RAGChunkMaxTokens int

	// GCS
	GCSBucket         string
	GCSEmbeddingsPath string

	// Document sources
	RAGDocsBasePath string
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
	f.String("openai_api_key", "", "OpenAI API key")
	f.String("openai_embedding_model", "text-embedding-3-small", "OpenAI embedding model")
	f.String("rag_llm_model", "gpt-4o", "LLM model for RAG answer generation")
	f.Int("rag_top_k", 5, "Number of chunks to retrieve")
	f.Int("rag_chunk_max_tokens", 800, "Maximum tokens per chunk")
	f.String("gcs_bucket", "", "GCS bucket for embeddings storage")
	f.String("gcs_embeddings_path", "rag/embeddings.gob", "GCS path for embeddings file")
	f.String("rag_docs_base_path", "", "Base path to document sources")

	bindings := map[string]string{
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"openai_api_key":            "OPENAI_API_KEY",
		"openai_embedding_model":    "OPENAI_EMBEDDING_MODEL",
		"rag_llm_model":             "RAG_LLM_MODEL",
		"rag_top_k":                 "RAG_TOP_K",
		"rag_chunk_max_tokens":      "RAG_CHUNK_MAX_TOKENS",
		"gcs_bucket":               "GCS_BUCKET",
		"gcs_embeddings_path":      "GCS_EMBEDDINGS_PATH",
		"rag_docs_base_path":       "RAG_DOCS_BASE_PATH",
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
			OpenAIAPIKey:            viper.GetString("openai_api_key"),
			OpenAIEmbeddingModel:    viper.GetString("openai_embedding_model"),
			RAGLLMModel:             viper.GetString("rag_llm_model"),
			RAGTopK:                 viper.GetInt("rag_top_k"),
			RAGChunkMaxTokens:       viper.GetInt("rag_chunk_max_tokens"),
			GCSBucket:               viper.GetString("gcs_bucket"),
			GCSEmbeddingsPath:       viper.GetString("gcs_embeddings_path"),
			RAGDocsBasePath:         viper.GetString("rag_docs_base_path"),
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
	if err = viper.BindPFlag("openai_api_key", cmd.Flags().Lookup("openai_api_key")); err != nil {
		return errors.Wrapf(err, "error binding openai_api_key flag")
	}
	if err = viper.BindPFlag("openai_embedding_model", cmd.Flags().Lookup("openai_embedding_model")); err != nil {
		return errors.Wrapf(err, "error binding openai_embedding_model flag")
	}
	if err = viper.BindPFlag("rag_llm_model", cmd.Flags().Lookup("rag_llm_model")); err != nil {
		return errors.Wrapf(err, "error binding rag_llm_model flag")
	}
	if err = viper.BindPFlag("rag_top_k", cmd.Flags().Lookup("rag_top_k")); err != nil {
		return errors.Wrapf(err, "error binding rag_top_k flag")
	}
	if err = viper.BindPFlag("rag_chunk_max_tokens", cmd.Flags().Lookup("rag_chunk_max_tokens")); err != nil {
		return errors.Wrapf(err, "error binding rag_chunk_max_tokens flag")
	}
	if err = viper.BindPFlag("gcs_bucket", cmd.Flags().Lookup("gcs_bucket")); err != nil {
		return errors.Wrapf(err, "error binding gcs_bucket flag")
	}
	if err = viper.BindPFlag("gcs_embeddings_path", cmd.Flags().Lookup("gcs_embeddings_path")); err != nil {
		return errors.Wrapf(err, "error binding gcs_embeddings_path flag")
	}
	if err = viper.BindPFlag("rag_docs_base_path", cmd.Flags().Lookup("rag_docs_base_path")); err != nil {
		return errors.Wrapf(err, "error binding rag_docs_base_path flag")
	}

	globalConfig = Config{
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),
		OpenAIAPIKey:            viper.GetString("openai_api_key"),
		OpenAIEmbeddingModel:    viper.GetString("openai_embedding_model"),
		RAGLLMModel:             viper.GetString("rag_llm_model"),
		RAGTopK:                 viper.GetInt("rag_top_k"),
		RAGChunkMaxTokens:       viper.GetInt("rag_chunk_max_tokens"),
		GCSBucket:               viper.GetString("gcs_bucket"),
		GCSEmbeddingsPath:       viper.GetString("gcs_embeddings_path"),
		RAGDocsBasePath:         viper.GetString("rag_docs_base_path"),
	}

	return nil
}
