package config

import (
	"fmt"
	"net/http"
	"strings"
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

	AIcallContactCaseRecreateRateLimitMinutes int // Rate limit window (minutes) after a contact_case-typed AIcall terminates during which recreation for the same reference_id is blocked (VOIP-1234).

	AIcallSendCooldownSeconds int // Minimum seconds between two Send() calls on the same AIcall, to bound LLM spend from rapid repeated sends.

	// Insight AI realtime call listening (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.12).
	AIcallListenEvaluateIntervalSeconds     int    // Debounce window. One listen evaluation turn per AIcall per this many seconds, regardless of how many sentences were spoken -- this is what decouples LLM cost from speech volume.
	AIcallListenWindowSize                  int    // Rolling transcript lines kept for continuity across turns and fed into each turn's context.
	AIcallListenQAContextSize               int    // Q&A message rows replayed into a listen turn's context, so the AI has continuity with what the agent asked.
	AIcallListenMaxTurnsPerAIcall           int    // Hard per-AIcall turn cap. Reaching it stops listening cleanly -- the backstop against a pathologically long call.
	AIcallListenBufferTTLHours              int    // TTL on the pending/window/lock/turn-count Redis keys. NOT the turn-id set, which has its own much shorter TTL below.
	AIcallListenTurnPipecatcallIDTTLSeconds int    // TTL on the registered listen-turn pipecatcall id set entries. Only needs to outlive one turn; generous headroom is cheap and self-expiring.
	AIcallListenDefaultLanguage             string // STT language used when the AIcall carries no STTLanguage.

	// The trigger path's own timings (design §5.1.1 step 7, §5.2.2, §5.12) --
	// they size waitForConfbridgeReady's bounded retry, the goroutine that
	// encloses it, and the per-AIcall create-or-reuse lock.
	//
	// ORDERING INVARIANT, pinned by Test_ListenConfigDefaults:
	//   ConfbridgeReadyMaxWait < EnsureGoroutineTimeout < StartLockTTL
	AIcallListenConfbridgeReadyPollIntervalSeconds int // Poll interval for waitForConfbridgeReady.
	AIcallListenConfbridgeReadyMaxWaitSeconds      int // Total wait budget before giving up with skipped_confbridge_not_ready. Must stay strictly less than AIcallListenEnsureGoroutineTimeoutSeconds below.
	AIcallListenEnsureGoroutineTimeoutSeconds      int // runListenStart's own detached-goroutine timeout -- purpose-built for this feature, not inherited from any other detached-goroutine pattern in this package.
	AIcallListenStartLockTTLSeconds                int // TTL on ai:listen:startlock:<aicall_id>, the per-AIcall lock serializing concurrent runListenStart create-or-reuse sequences. Must strictly EXCEED AIcallListenEnsureGoroutineTimeoutSeconds so it can never expire under a goroutine still working inside its own budget -- NOT derived by summing the RPC timeouts inside the lock, a derivation the design tried and withdrew.
	AIcallListenStartLockReleaseTimeoutSeconds     int // Bound on the DETACHED context (context.WithTimeout(context.WithoutCancel(ctx), ...)) the lock's Release call runs under, so a stuck Redis call during cleanup cannot hang the releasing goroutine. Independent of, and far below, the TTL above.

	// Insight AI realtime listening on conversation (message) Cases
	// (docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.12).
	AIcallListenConversationMaxMessageChars int // Per-field cap (subject, text and the joined media tokens are each capped, so one message contributes at most about three times this many characters) applied before a conversation line is buffered; the window is line-counted, not byte-counted, so an email body must not be allowed to dominate it.
	AIcallListenConversationFlushJitterMs   int // Upper bound of the random jitter added to the deferred flush delay, so two replicas' timers for one AIcall do not race the debounce lock at the same instant.

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
	f.Int("aicall_contact_case_recreate_rate_limit_minutes", 5, "Rate limit window (minutes) after a contact_case-typed AIcall terminates during which recreation for the same reference_id is blocked")
	f.Int("aicall_send_cooldown_seconds", 3, "Minimum seconds between two Send() calls on the same AIcall")
	f.Int("aicall_listen_evaluate_interval_seconds", 20, "Debounce window (seconds) between Insight AI listen evaluation turns on one AIcall")
	f.Int("aicall_listen_window_size", 40, "Rolling transcript lines kept in a listen turn's context")
	f.Int("aicall_listen_qa_context_size", 10, "Q&A message rows replayed into a listen turn's context")
	f.Int("aicall_listen_max_turns_per_aicall", 60, "Hard cap on listen evaluation turns per AIcall")
	f.Int("aicall_listen_buffer_ttl_hours", 6, "TTL (hours) on the listen buffer, lock and turn-count Redis keys")
	f.Int("aicall_listen_turn_pipecatcall_id_ttl_seconds", 180, "TTL (seconds) on registered listen-turn pipecatcall id set entries")
	f.String("aicall_listen_default_language", "en-US", "STT language for listening when the AIcall has none set")
	f.Int("aicall_listen_confbridge_ready_poll_interval_seconds", 2, "Poll interval (seconds) for the bounded confbridge-readiness retry before listening starts")
	f.Int("aicall_listen_confbridge_ready_max_wait_seconds", 30, "Total wait budget (seconds) for the confbridge-readiness retry before giving up")
	f.Int("aicall_listen_ensure_goroutine_timeout_seconds", 45, "Timeout (seconds) for runListenStart's own detached goroutine; must stay strictly greater than aicall_listen_confbridge_ready_max_wait_seconds")
	f.Int("aicall_listen_start_lock_ttl_seconds", 60, "TTL (seconds) on the per-AIcall listen-start lock; must stay strictly greater than aicall_listen_ensure_goroutine_timeout_seconds")
	f.Int("aicall_listen_start_lock_release_timeout_seconds", 3, "Timeout (seconds) on the detached context the listen-start lock's release runs under")
	f.Int("aicall_listen_conversation_max_message_chars", 2000, "Per-field character cap (subject, text and the joined media tokens each) applied before a conversation line is buffered for listening")
	f.Int("aicall_listen_conversation_flush_jitter_ms", 1000, "Upper bound (milliseconds) of the random jitter added to the conversation listen deferred-flush delay")
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

		"aicall_conversation_idle_timeout_hours":          "AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS",
		"aicall_contact_case_recreate_rate_limit_minutes": "AICALL_CONTACT_CASE_RECREATE_RATE_LIMIT_MINUTES",
		"aicall_send_cooldown_seconds":                    "AICALL_SEND_COOLDOWN_SECONDS",

		"aicall_listen_evaluate_interval_seconds":              "AICALL_LISTEN_EVALUATE_INTERVAL_SECONDS",
		"aicall_listen_window_size":                            "AICALL_LISTEN_WINDOW_SIZE",
		"aicall_listen_qa_context_size":                        "AICALL_LISTEN_QA_CONTEXT_SIZE",
		"aicall_listen_max_turns_per_aicall":                   "AICALL_LISTEN_MAX_TURNS_PER_AICALL",
		"aicall_listen_buffer_ttl_hours":                       "AICALL_LISTEN_BUFFER_TTL_HOURS",
		"aicall_listen_turn_pipecatcall_id_ttl_seconds":        "AICALL_LISTEN_TURN_PIPECATCALL_ID_TTL_SECONDS",
		"aicall_listen_default_language":                       "AICALL_LISTEN_DEFAULT_LANGUAGE",
		"aicall_listen_confbridge_ready_poll_interval_seconds": "AICALL_LISTEN_CONFBRIDGE_READY_POLL_INTERVAL_SECONDS",
		"aicall_listen_confbridge_ready_max_wait_seconds":      "AICALL_LISTEN_CONFBRIDGE_READY_MAX_WAIT_SECONDS",
		"aicall_listen_ensure_goroutine_timeout_seconds":       "AICALL_LISTEN_ENSURE_GOROUTINE_TIMEOUT_SECONDS",
		"aicall_listen_start_lock_ttl_seconds":                 "AICALL_LISTEN_START_LOCK_TTL_SECONDS",
		"aicall_listen_start_lock_release_timeout_seconds":     "AICALL_LISTEN_START_LOCK_RELEASE_TIMEOUT_SECONDS",

		"aicall_listen_conversation_max_message_chars": "AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS",
		"aicall_listen_conversation_flush_jitter_ms":   "AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS",

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

			AIcallContactCaseRecreateRateLimitMinutes: viper.GetInt("aicall_contact_case_recreate_rate_limit_minutes"),

			AIcallSendCooldownSeconds: viper.GetInt("aicall_send_cooldown_seconds"),

			AIcallListenEvaluateIntervalSeconds:     viper.GetInt("aicall_listen_evaluate_interval_seconds"),
			AIcallListenWindowSize:                  viper.GetInt("aicall_listen_window_size"),
			AIcallListenQAContextSize:               viper.GetInt("aicall_listen_qa_context_size"),
			AIcallListenMaxTurnsPerAIcall:           viper.GetInt("aicall_listen_max_turns_per_aicall"),
			AIcallListenBufferTTLHours:              viper.GetInt("aicall_listen_buffer_ttl_hours"),
			AIcallListenTurnPipecatcallIDTTLSeconds: viper.GetInt("aicall_listen_turn_pipecatcall_id_ttl_seconds"),
			AIcallListenDefaultLanguage:             viper.GetString("aicall_listen_default_language"),

			AIcallListenConfbridgeReadyPollIntervalSeconds: viper.GetInt("aicall_listen_confbridge_ready_poll_interval_seconds"),
			AIcallListenConfbridgeReadyMaxWaitSeconds:      viper.GetInt("aicall_listen_confbridge_ready_max_wait_seconds"),
			AIcallListenEnsureGoroutineTimeoutSeconds:      viper.GetInt("aicall_listen_ensure_goroutine_timeout_seconds"),
			AIcallListenStartLockTTLSeconds:                viper.GetInt("aicall_listen_start_lock_ttl_seconds"),
			AIcallListenStartLockReleaseTimeoutSeconds:     viper.GetInt("aicall_listen_start_lock_release_timeout_seconds"),

			AIcallListenConversationMaxMessageChars: viper.GetInt("aicall_listen_conversation_max_message_chars"),
			AIcallListenConversationFlushJitterMs:   viper.GetInt("aicall_listen_conversation_flush_jitter_ms"),

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

// SetAIcallContactCaseRecreateRateLimitMinutesForTest overrides the contact_case
// recreate rate limit window in the global config without going through the
// Bootstrap+LoadGlobalConfig path.
// USE ONLY FROM TESTS.
func SetAIcallContactCaseRecreateRateLimitMinutesForTest(minutes int) {
	globalConfig.AIcallContactCaseRecreateRateLimitMinutes = minutes
}

// SetAIcallSendCooldownSecondsForTest overrides the send cooldown in tests.
func SetAIcallSendCooldownSecondsForTest(seconds int) {
	globalConfig.AIcallSendCooldownSeconds = seconds
}

// Validate checks the loaded configuration for values that would make the
// service misbehave at runtime, and returns an error naming every offending
// value. Call it AFTER LoadGlobalConfig and BEFORE anything starts serving.
//
// IT FAILS THE PROCESS, IT DOES NOT CLAMP (review round 1 finding MEDIUM-1).
// Every condition below is a deploy-time typo in an env var, not a state a
// running system can drift into -- silently substituting a "sensible" value
// would leave the operator's own configuration disagreeing with the process's
// actual behaviour, which is strictly harder to diagnose than a refused start.
func Validate() error {
	if errListen := validateListenConfig(); errListen != nil {
		return errListen
	}

	return nil
}

// validateListenConfig enforces the Insight AI listen flags' two standing
// invariants.
//
// (1) EVERY TIMING AND SIZING VALUE IS STRICTLY POSITIVE. Four of these are
// actively dangerous at zero, and none is caught anywhere else:
// AICALL_LISTEN_CONFBRIDGE_READY_POLL_INTERVAL_SECONDS=0 turns
// waitForConfbridgeReady's select into a busy-loop that spins one pair of RPCs
// per iteration for the whole wait budget;
// AICALL_LISTEN_START_LOCK_RELEASE_TIMEOUT_SECONDS=0 makes every lock release
// run on an already-expired context, so EVERY release silently no-ops and every
// per-AIcall start lock strands for its full TTL;
// AICALL_LISTEN_WINDOW_SIZE=0 makes the `LTRIM key 0 -1` inside the
// window-push Lua script a no-op, so the rolling window grows unbounded on the
// platform-wide transcript-intake hot path for the whole buffer TTL (and a
// NEGATIVE value trims from the wrong end, keeping the OLDEST lines instead of
// the newest); and AICALL_LISTEN_MAX_TURNS_PER_AICALL=0 makes the very first
// turn exceed the cap, silently disabling listening turns outright (review
// round 2 finding MEDIUM-4).
//
// (2) THE ORDERING ConfbridgeReadyMaxWait < EnsureGoroutineTimeout <
// StartLockTTL. runListenStart's confbridge poll runs INSIDE the goroutine
// timeout and needs headroom for the RPCs each poll makes, and the start lock
// must outlive the goroutine that holds it or the lock can expire under a
// goroutine still legitimately working (which is precisely the clobbering the
// lock exists to prevent). Test_ListenConfigDefaults asserts this ordering, but
// only against values the test itself sets -- it structurally cannot catch a
// real environment-variable override, which is what this function is for.
//
// It runs unconditionally at startup: a deploy carrying a broken timing value
// is a deploy that breaks the moment a listen session runs, and finding that
// out at rollout time is the whole failure this prevents.
func validateListenConfig() error {
	positives := []struct {
		name  string
		value int
	}{
		{"aicall_listen_evaluate_interval_seconds", globalConfig.AIcallListenEvaluateIntervalSeconds},
		{"aicall_listen_window_size", globalConfig.AIcallListenWindowSize},
		{"aicall_listen_qa_context_size", globalConfig.AIcallListenQAContextSize},
		{"aicall_listen_max_turns_per_aicall", globalConfig.AIcallListenMaxTurnsPerAIcall},
		{"aicall_listen_buffer_ttl_hours", globalConfig.AIcallListenBufferTTLHours},
		{"aicall_listen_turn_pipecatcall_id_ttl_seconds", globalConfig.AIcallListenTurnPipecatcallIDTTLSeconds},
		{"aicall_listen_confbridge_ready_poll_interval_seconds", globalConfig.AIcallListenConfbridgeReadyPollIntervalSeconds},
		{"aicall_listen_confbridge_ready_max_wait_seconds", globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds},
		{"aicall_listen_ensure_goroutine_timeout_seconds", globalConfig.AIcallListenEnsureGoroutineTimeoutSeconds},
		{"aicall_listen_start_lock_ttl_seconds", globalConfig.AIcallListenStartLockTTLSeconds},
		{"aicall_listen_start_lock_release_timeout_seconds", globalConfig.AIcallListenStartLockReleaseTimeoutSeconds},
		{"aicall_listen_conversation_max_message_chars", globalConfig.AIcallListenConversationMaxMessageChars},
	}

	invalid := []string{}
	for _, p := range positives {
		if p.value <= 0 {
			invalid = append(invalid, fmt.Sprintf("%s must be > 0, got %d", p.name, p.value))
		}
	}
	if len(invalid) > 0 {
		return errors.Errorf("invalid listen configuration: %s", strings.Join(invalid, "; "))
	}

	if globalConfig.AIcallListenConversationFlushJitterMs < 0 {
		return errors.Errorf("invalid listen configuration: aicall_listen_conversation_flush_jitter_ms must be >= 0, got %d", globalConfig.AIcallListenConversationFlushJitterMs)
	}

	maxWait := globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds
	goroutineTimeout := globalConfig.AIcallListenEnsureGoroutineTimeoutSeconds
	lockTTL := globalConfig.AIcallListenStartLockTTLSeconds
	if maxWait >= goroutineTimeout || goroutineTimeout >= lockTTL {
		return errors.Errorf(
			"invalid listen configuration: aicall_listen_confbridge_ready_max_wait_seconds (%d) < aicall_listen_ensure_goroutine_timeout_seconds (%d) < aicall_listen_start_lock_ttl_seconds (%d) must hold",
			maxWait, goroutineTimeout, lockTTL,
		)
	}

	return nil
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

// SetListenDefaultsForTest populates the Insight AI listen flags in the global
// config with their shipped defaults, without going through the
// Bootstrap+LoadGlobalConfig path (LoadGlobalConfig is sync.Once-guarded, so a
// test cannot re-run it).
// USE ONLY FROM TESTS.
func SetListenDefaultsForTest() {
	globalConfig.AIcallListenEvaluateIntervalSeconds = 20
	globalConfig.AIcallListenWindowSize = 40
	globalConfig.AIcallListenQAContextSize = 10
	globalConfig.AIcallListenMaxTurnsPerAIcall = 60
	globalConfig.AIcallListenBufferTTLHours = 6
	globalConfig.AIcallListenTurnPipecatcallIDTTLSeconds = 180
	globalConfig.AIcallListenDefaultLanguage = "en-US"
	globalConfig.AIcallListenConfbridgeReadyPollIntervalSeconds = 2
	globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds = 30
	globalConfig.AIcallListenEnsureGoroutineTimeoutSeconds = 45
	globalConfig.AIcallListenStartLockTTLSeconds = 60
	globalConfig.AIcallListenStartLockReleaseTimeoutSeconds = 3
	globalConfig.AIcallListenConversationMaxMessageChars = 2000
	globalConfig.AIcallListenConversationFlushJitterMs = 1000
}

// SetAIcallListenStartLockTTLForTest overrides the per-AIcall listen-start
// lock's TTL in tests, so Task 20's "simulated crash, lock held for the full
// TTL" row does not have to wait 60 real seconds.
// USE ONLY FROM TESTS.
func SetAIcallListenStartLockTTLForTest(seconds int) {
	globalConfig.AIcallListenStartLockTTLSeconds = seconds
}

// SetAIcallListenConfbridgeReadyPollIntervalForTest overrides
// waitForConfbridgeReady's poll interval in tests. Combine with
// SetAIcallListenConfbridgeReadyMaxWaitForTest to keep the timeout-driven
// rows of Test_waitForConfbridgeReady fast.
// USE ONLY FROM TESTS.
func SetAIcallListenConfbridgeReadyPollIntervalForTest(seconds int) {
	globalConfig.AIcallListenConfbridgeReadyPollIntervalSeconds = seconds
}

// SetAIcallListenConfbridgeReadyMaxWaitForTest overrides
// waitForConfbridgeReady's wait budget in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenConfbridgeReadyMaxWaitForTest(seconds int) {
	globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds = seconds
}

// SetAIcallListenMaxTurnsPerAIcallForTest overrides the per-AIcall turn cap in
// tests, so a cap-exceeded path can be exercised without running 60 turns.
// USE ONLY FROM TESTS.
func SetAIcallListenMaxTurnsPerAIcallForTest(turns int) {
	globalConfig.AIcallListenMaxTurnsPerAIcall = turns
}

// SetAIcallListenWindowSizeForTest overrides the rolling transcript window size
// in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenWindowSizeForTest(size int) {
	globalConfig.AIcallListenWindowSize = size
}

// SetAIcallListenQAContextSizeForTest overrides the Q&A context row budget in
// tests.
// USE ONLY FROM TESTS.
func SetAIcallListenQAContextSizeForTest(size int) {
	globalConfig.AIcallListenQAContextSize = size
}

// SetAIcallListenConversationMaxMessageCharsForTest overrides the per-message
// truncation cap in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenConversationMaxMessageCharsForTest(chars int) {
	globalConfig.AIcallListenConversationMaxMessageChars = chars
}

// SetAIcallListenConversationFlushJitterMsForTest overrides the deferred-flush
// jitter bound in tests (0 makes the delay deterministic).
// USE ONLY FROM TESTS.
func SetAIcallListenConversationFlushJitterMsForTest(ms int) {
	globalConfig.AIcallListenConversationFlushJitterMs = ms
}
