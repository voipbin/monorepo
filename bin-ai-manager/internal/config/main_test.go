package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				RabbitMQAddress:                    "amqp://guest:guest@localhost:5672",
				PrometheusEndpoint:                 "/metrics",
				PrometheusListenAddress:            ":2112",
				DatabaseDSN:                        "user:pass@tcp(127.0.0.1:3306)/db",
				RedisAddress:                       "127.0.0.1:6379",
				RedisPassword:                      "secret",
				RedisDatabase:                      1,
				EngineKeyChatGPT:                   "sk-test-key",
				GoogleAPIKey:                       "AIza-test-key",
				AIcallConversationIdleTimeoutHours: 48,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = tt.setupConfig

			res := Get()

			if res.RabbitMQAddress != tt.setupConfig.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.setupConfig.RabbitMQAddress, res.RabbitMQAddress)
			}
			if res.PrometheusEndpoint != tt.setupConfig.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.setupConfig.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.setupConfig.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.setupConfig.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.DatabaseDSN != tt.setupConfig.DatabaseDSN {
				t.Errorf("Wrong DatabaseDSN. expect: %s, got: %s", tt.setupConfig.DatabaseDSN, res.DatabaseDSN)
			}
			if res.RedisAddress != tt.setupConfig.RedisAddress {
				t.Errorf("Wrong RedisAddress. expect: %s, got: %s", tt.setupConfig.RedisAddress, res.RedisAddress)
			}
			if res.RedisPassword != tt.setupConfig.RedisPassword {
				t.Errorf("Wrong RedisPassword. expect: %s, got: %s", tt.setupConfig.RedisPassword, res.RedisPassword)
			}
			if res.RedisDatabase != tt.setupConfig.RedisDatabase {
				t.Errorf("Wrong RedisDatabase. expect: %d, got: %d", tt.setupConfig.RedisDatabase, res.RedisDatabase)
			}
			if res.EngineKeyChatGPT != tt.setupConfig.EngineKeyChatGPT {
				t.Errorf("Wrong EngineKeyChatGPT. expect: %s, got: %s", tt.setupConfig.EngineKeyChatGPT, res.EngineKeyChatGPT)
			}
			if res.GoogleAPIKey != tt.setupConfig.GoogleAPIKey {
				t.Errorf("Wrong GoogleAPIKey. expect: %s, got: %s", tt.setupConfig.GoogleAPIKey, res.GoogleAPIKey)
			}
			if res.AIcallConversationIdleTimeoutHours != tt.setupConfig.AIcallConversationIdleTimeoutHours {
				t.Errorf("Wrong AIcallConversationIdleTimeoutHours. expect: %d, got: %d", tt.setupConfig.AIcallConversationIdleTimeoutHours, res.AIcallConversationIdleTimeoutHours)
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name string

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			err := Bootstrap(rootCmd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify flags were registered
			if rootCmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
				t.Errorf("Expected prometheus_endpoint flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
				t.Errorf("Expected prometheus_listen_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("database_dsn") == nil {
				t.Errorf("Expected database_dsn flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_address") == nil {
				t.Errorf("Expected redis_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_password") == nil {
				t.Errorf("Expected redis_password flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("redis_database") == nil {
				t.Errorf("Expected redis_database flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("engine_key_chatgpt") == nil {
				t.Errorf("Expected engine_key_chatgpt flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("google_api_key") == nil {
				t.Errorf("Expected google_api_key flag to be registered")
			}
			flagAIcallIdle := rootCmd.PersistentFlags().Lookup("aicall_conversation_idle_timeout_hours")
			if flagAIcallIdle == nil {
				t.Errorf("Expected aicall_conversation_idle_timeout_hours flag to be registered")
			} else if flagAIcallIdle.DefValue != "24" {
				t.Errorf("Wrong aicall_conversation_idle_timeout_hours default. expect: 24, got: %s", flagAIcallIdle.DefValue)
			}

			flagAnalysisModel := rootCmd.PersistentFlags().Lookup("analysis_default_model")
			if flagAnalysisModel == nil {
				t.Errorf("Expected analysis_default_model flag to be registered")
			} else if flagAnalysisModel.DefValue != "gemini-2.5-flash" {
				t.Errorf("Wrong analysis_default_model default. expect: gemini-2.5-flash, got: %s", flagAnalysisModel.DefValue)
			}
			flagAnalysisAllowed := rootCmd.PersistentFlags().Lookup("analysis_allowed_models")
			if flagAnalysisAllowed == nil {
				t.Errorf("Expected analysis_allowed_models flag to be registered")
			} else if flagAnalysisAllowed.DefValue != "gemini-2.5-flash,gemini-2.5-pro" {
				t.Errorf("Wrong analysis_allowed_models default. expect: gemini-2.5-flash,gemini-2.5-pro, got: %s", flagAnalysisAllowed.DefValue)
			}
			flagAnalysisBaseURL := rootCmd.PersistentFlags().Lookup("analysis_engine_base_url")
			if flagAnalysisBaseURL == nil {
				t.Errorf("Expected analysis_engine_base_url flag to be registered")
			} else if flagAnalysisBaseURL.DefValue != "https://generativelanguage.googleapis.com/v1beta/openai/" {
				t.Errorf("Wrong analysis_engine_base_url default. got: %s", flagAnalysisBaseURL.DefValue)
			}
			flagAnalysisReasoning := rootCmd.PersistentFlags().Lookup("analysis_reasoning_effort")
			if flagAnalysisReasoning == nil {
				t.Errorf("Expected analysis_reasoning_effort flag to be registered")
			} else if flagAnalysisReasoning.DefValue != "none" {
				t.Errorf("Wrong analysis_reasoning_effort default. expect: none, got: %s", flagAnalysisReasoning.DefValue)
			}
			flagAnalysisMaxOut := rootCmd.PersistentFlags().Lookup("analysis_max_output_tokens")
			if flagAnalysisMaxOut == nil {
				t.Errorf("Expected analysis_max_output_tokens flag to be registered")
			} else if flagAnalysisMaxOut.DefValue != "16384" {
				t.Errorf("Wrong analysis_max_output_tokens default. expect: 16384, got: %s", flagAnalysisMaxOut.DefValue)
			}
		})
	}
}

func TestSetAIcallConversationIdleTimeoutHoursForTest(t *testing.T) {
	tests := []struct {
		name string

		hours int
	}{
		{
			name: "overrides_idle_timeout",

			hours: 72,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = Config{}

			SetAIcallConversationIdleTimeoutHoursForTest(tt.hours)

			res := Get()
			if res.AIcallConversationIdleTimeoutHours != tt.hours {
				t.Errorf("Wrong AIcallConversationIdleTimeoutHours. expect: %d, got: %d", tt.hours, res.AIcallConversationIdleTimeoutHours)
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loads_global_config_only_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once for test isolation
			once = sync.Once{}

			// First call should work without panic
			LoadGlobalConfig()

			// Second call should be a no-op due to sync.Once
			LoadGlobalConfig()

			// Verify Get returns a valid pointer
			cfg := Get()
			if cfg == nil {
				t.Errorf("Expected non-nil config from Get()")
			}
		})
	}
}

// Test_ListenConfigDefaults pins the shipped defaults for the Insight AI
// realtime-listen flags. The first assertion is the important one: the feature
// must ship dark. The rest guard against a zero-value default silently
// disabling the debounce (interval 0 = a turn per transcript segment, the exact
// unbounded-cost shape the design exists to avoid) or the turn cap.
func Test_ListenConfigDefaults(t *testing.T) {
	SetListenDefaultsForTest()

	cfg := Get()

	if cfg.AIcallListenEnabled {
		t.Errorf("AIcallListenEnabled must default to false -- the feature ships dark")
	}
	if cfg.AIcallListenEvaluateIntervalSeconds != 20 {
		t.Errorf("AIcallListenEvaluateIntervalSeconds mismatch. expected: 20, got: %d", cfg.AIcallListenEvaluateIntervalSeconds)
	}
	if cfg.AIcallListenWindowSize != 40 {
		t.Errorf("AIcallListenWindowSize mismatch. expected: 40, got: %d", cfg.AIcallListenWindowSize)
	}
	if cfg.AIcallListenQAContextSize != 10 {
		t.Errorf("AIcallListenQAContextSize mismatch. expected: 10, got: %d", cfg.AIcallListenQAContextSize)
	}
	if cfg.AIcallListenMaxTurnsPerAIcall != 60 {
		t.Errorf("AIcallListenMaxTurnsPerAIcall mismatch. expected: 60, got: %d", cfg.AIcallListenMaxTurnsPerAIcall)
	}
	if cfg.AIcallListenBufferTTLHours != 6 {
		t.Errorf("AIcallListenBufferTTLHours mismatch. expected: 6, got: %d", cfg.AIcallListenBufferTTLHours)
	}
	if cfg.AIcallListenTurnPipecatcallIDTTLSeconds != 180 {
		t.Errorf("AIcallListenTurnPipecatcallIDTTLSeconds mismatch. expected: 180, got: %d", cfg.AIcallListenTurnPipecatcallIDTTLSeconds)
	}
	if cfg.AIcallListenDefaultLanguage != "en-US" {
		t.Errorf("AIcallListenDefaultLanguage mismatch. expected: %q, got: %q", "en-US", cfg.AIcallListenDefaultLanguage)
	}
	if cfg.AIcallListenConfbridgeReadyPollIntervalSeconds != 2 {
		t.Errorf("AIcallListenConfbridgeReadyPollIntervalSeconds mismatch. expected: 2, got: %d", cfg.AIcallListenConfbridgeReadyPollIntervalSeconds)
	}
	if cfg.AIcallListenConfbridgeReadyMaxWaitSeconds != 30 {
		t.Errorf("AIcallListenConfbridgeReadyMaxWaitSeconds mismatch. expected: 30, got: %d", cfg.AIcallListenConfbridgeReadyMaxWaitSeconds)
	}
	if cfg.AIcallListenEnsureGoroutineTimeoutSeconds != 45 {
		t.Errorf("AIcallListenEnsureGoroutineTimeoutSeconds mismatch. expected: 45, got: %d", cfg.AIcallListenEnsureGoroutineTimeoutSeconds)
	}
	if cfg.AIcallListenStartLockTTLSeconds != 60 {
		t.Errorf("AIcallListenStartLockTTLSeconds mismatch. expected: 60, got: %d", cfg.AIcallListenStartLockTTLSeconds)
	}
	if cfg.AIcallListenStartLockReleaseTimeoutSeconds != 3 {
		t.Errorf("AIcallListenStartLockReleaseTimeoutSeconds mismatch. expected: 3, got: %d", cfg.AIcallListenStartLockReleaseTimeoutSeconds)
	}
	// The goroutine timeout must have headroom over the max-wait budget it
	// encloses -- pinned here as a standing invariant, not just a one-time
	// default check, since the two are set independently.
	if cfg.AIcallListenEnsureGoroutineTimeoutSeconds <= cfg.AIcallListenConfbridgeReadyMaxWaitSeconds {
		t.Errorf("AIcallListenEnsureGoroutineTimeoutSeconds (%d) must be strictly greater than AIcallListenConfbridgeReadyMaxWaitSeconds (%d)", cfg.AIcallListenEnsureGoroutineTimeoutSeconds, cfg.AIcallListenConfbridgeReadyMaxWaitSeconds)
	}
	// And the start lock's TTL must in turn exceed that goroutine timeout
	// (design §5.2.2, review round 15 finding HIGH-1): the lock must never be
	// able to expire out from under a goroutine that is still working inside
	// its own legitimate budget, because a second goroutine acquiring it would
	// reopen exactly the same-AIcall clobbering race the lock exists to close.
	if cfg.AIcallListenStartLockTTLSeconds <= cfg.AIcallListenEnsureGoroutineTimeoutSeconds {
		t.Errorf("AIcallListenStartLockTTLSeconds (%d) must be strictly greater than AIcallListenEnsureGoroutineTimeoutSeconds (%d)", cfg.AIcallListenStartLockTTLSeconds, cfg.AIcallListenEnsureGoroutineTimeoutSeconds)
	}
	// The release bound is a small, independent timeout on the DETACHED
	// context the lock's Release call runs under -- it must stay far below the
	// TTL, and must never be conflated with it.
	if cfg.AIcallListenStartLockReleaseTimeoutSeconds >= cfg.AIcallListenStartLockTTLSeconds {
		t.Errorf("AIcallListenStartLockReleaseTimeoutSeconds (%d) must stay well below AIcallListenStartLockTTLSeconds (%d)", cfg.AIcallListenStartLockReleaseTimeoutSeconds, cfg.AIcallListenStartLockTTLSeconds)
	}
}
