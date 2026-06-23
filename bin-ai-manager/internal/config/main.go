package config

import (
	"net/http"
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":8080").
	DatabaseDSN             string // DatabaseDSN is the data source name used to connect to the primary database.
	RedisAddress            string // RedisAddress is the address (including host and port) of the Redis server.
	RedisPassword           string // RedisPassword is the password used for authenticating to the Redis server.
	RedisDatabase           int    // RedisDatabase is the numeric Redis logical database index to select, not a name.
	EngineKeyChatGPT        string // EngineKeyChatGPT is the API key for ChatGPT engine.
	GoogleAPIKey            string // GoogleAPIKey is the Google API key used for Gemini audit evaluation.

	AIcallConversationIdleTimeoutHours int // Idle timeout (hours) after which a conversation-typed AIcall is treated as expired and a new one is created on the next inbound message.

	AnalysisDefaultModel    string // AnalysisDefaultModel is the default model for the generic analysis gateway.
	AnalysisAllowedModels   string // AnalysisAllowedModels is a comma-separated allow-set of models the analysis gateway accepts.
	AnalysisEngineBaseURL   string // AnalysisEngineBaseURL is the base URL for the analysis gateway LLM engine (Gemini OpenAI-compat by default; clear to use OpenAI).
	AnalysisReasoningEffort string // AnalysisReasoningEffort is sent as reasoning_effort on the analysis gateway request ("none" disables Gemini thinking; empty omits the field).
	AnalysisMaxInputBytes   int    // AnalysisMaxInputBytes caps the prompt+data byte size accepted by the analysis gateway.
	AnalysisMaxOutputTokens int    // AnalysisMaxOutputTokens caps the output tokens of the analysis gateway (runaway guard).
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
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")
	f.String("engine_key_chatgpt", "", "Engine key for chatgpt")
	f.String("google_api_key", "", "Google API key for Gemini audit evaluation")
	f.Int("aicall_conversation_idle_timeout_hours", 24, "Idle timeout (hours) for conversation-typed AIcalls before they expire")
	f.String("analysis_default_model", "gemini-2.5-flash", "Default model for the generic analysis gateway")
	f.String("analysis_allowed_models", "gemini-2.5-flash,gemini-2.5-pro", "Comma-separated allow-set of models for the analysis gateway")
	f.String("analysis_engine_base_url", "https://generativelanguage.googleapis.com/v1beta/openai/", "Base URL for the analysis gateway LLM engine (Gemini OpenAI-compat by default; clear to use OpenAI)")
	f.String("analysis_reasoning_effort", "none", "reasoning_effort for the analysis gateway (none disables Gemini thinking; empty omits the field)")
	f.Int("analysis_max_input_bytes", 262144, "Max prompt+data bytes accepted by the analysis gateway")
	f.Int("analysis_max_output_tokens", 16384, "Max output tokens for the analysis gateway (runaway guard)")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"engine_key_chatgpt":        "ENGINE_KEY_CHATGPT",
		"google_api_key":            "GOOGLE_API_KEY",

		"aicall_conversation_idle_timeout_hours": "AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS",

		"analysis_default_model":     "ANALYSIS_DEFAULT_MODEL",
		"analysis_allowed_models":    "ANALYSIS_ALLOWED_MODELS",
		"analysis_engine_base_url":   "ANALYSIS_ENGINE_BASE_URL",
		"analysis_reasoning_effort":  "ANALYSIS_REASONING_EFFORT",
		"analysis_max_input_bytes":   "ANALYSIS_MAX_INPUT_BYTES",
		"analysis_max_output_tokens": "ANALYSIS_MAX_OUTPUT_TOKENS",
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
			EngineKeyChatGPT:        viper.GetString("engine_key_chatgpt"),
			GoogleAPIKey:            viper.GetString("google_api_key"),

			AIcallConversationIdleTimeoutHours: viper.GetInt("aicall_conversation_idle_timeout_hours"),

			AnalysisDefaultModel:    viper.GetString("analysis_default_model"),
			AnalysisAllowedModels:   viper.GetString("analysis_allowed_models"),
			AnalysisEngineBaseURL:   viper.GetString("analysis_engine_base_url"),
			AnalysisReasoningEffort: viper.GetString("analysis_reasoning_effort"),
			AnalysisMaxInputBytes:   viper.GetInt("analysis_max_input_bytes"),
			AnalysisMaxOutputTokens: viper.GetInt("analysis_max_output_tokens"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// SetAIcallConversationIdleTimeoutHoursForTest overrides the idle timeout in
// the global config without going through the Bootstrap+LoadGlobalConfig path.
// USE ONLY FROM TESTS.
func SetAIcallConversationIdleTimeoutHoursForTest(hours int) {
	globalConfig.AIcallConversationIdleTimeoutHours = hours
}

// InitPrometheus initializes Prometheus metrics server.
// Must be called AFTER LoadGlobalConfig().
func InitPrometheus() {
	cfg := Get()

	// Skip Prometheus initialization if endpoint or listen address is not configured
	if cfg.PrometheusEndpoint == "" || cfg.PrometheusListenAddress == "" {
		logrus.Debug("Prometheus metrics server disabled (endpoint or listen address not configured)")
		return
	}

	http.Handle(cfg.PrometheusEndpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", cfg.PrometheusListenAddress, cfg.PrometheusEndpoint)
		if err := http.ListenAndServe(cfg.PrometheusListenAddress, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}
