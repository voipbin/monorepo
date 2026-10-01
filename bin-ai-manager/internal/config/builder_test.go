package config

import (
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The Assistant Builder is off by default. A deploy that never mentions it must
// behave exactly as before this feature existed.
func Test_BuilderConfigDefaults(t *testing.T) {
	viper.Reset()
	cmd := &cobra.Command{}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	tests := []struct {
		key    string
		expect interface{}
	}{
		{"ai_builder_enabled", false},
		{"ai_builder_model", "gemini-3.8-flash"},
		{"ai_builder_reasoning_effort", "none"},
		{"ai_builder_max_output_tokens", 4096},
		{"ai_builder_daily_limit", 200},
		{"ai_builder_max_concurrent", 3},
		{"ai_builder_llm_timeout_seconds", 40},
	}
	for _, tt := range tests {
		got := viper.Get(tt.key)
		switch want := tt.expect.(type) {
		case bool:
			if viper.GetBool(tt.key) != want {
				t.Errorf("%s: expected %v, got %v", tt.key, want, got)
			}
		case int:
			if viper.GetInt(tt.key) != want {
				t.Errorf("%s: expected %v, got %v", tt.key, want, got)
			}
		case string:
			if viper.GetString(tt.key) != want {
				t.Errorf("%s: expected %v, got %v", tt.key, want, got)
			}
		}
	}
}

// The environment variable names are the deploy-time contract.
func Test_BuilderConfigEnvBindings(t *testing.T) {
	viper.Reset()
	cmd := &cobra.Command{}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	t.Setenv("AI_BUILDER_ENABLED", "true")
	t.Setenv("AI_BUILDER_MODEL", "some-other-model")
	t.Setenv("AI_BUILDER_REASONING_EFFORT", "low")
	t.Setenv("AI_BUILDER_MAX_OUTPUT_TOKENS", "1234")
	t.Setenv("AI_BUILDER_DAILY_LIMIT", "55")
	t.Setenv("AI_BUILDER_MAX_CONCURRENT", "7")
	t.Setenv("AI_BUILDER_LLM_TIMEOUT_SECONDS", "21")

	if !viper.GetBool("ai_builder_enabled") {
		t.Error("AI_BUILDER_ENABLED not bound")
	}
	if viper.GetString("ai_builder_model") != "some-other-model" {
		t.Error("AI_BUILDER_MODEL not bound")
	}
	if viper.GetString("ai_builder_reasoning_effort") != "low" {
		t.Error("AI_BUILDER_REASONING_EFFORT not bound")
	}
	if viper.GetInt("ai_builder_max_output_tokens") != 1234 {
		t.Error("AI_BUILDER_MAX_OUTPUT_TOKENS not bound")
	}
	if viper.GetInt("ai_builder_daily_limit") != 55 {
		t.Error("AI_BUILDER_DAILY_LIMIT not bound")
	}
	if viper.GetInt("ai_builder_max_concurrent") != 7 {
		t.Error("AI_BUILDER_MAX_CONCURRENT not bound")
	}
	if viper.GetInt("ai_builder_llm_timeout_seconds") != 21 {
		t.Error("AI_BUILDER_LLM_TIMEOUT_SECONDS not bound")
	}
}

// Validate must refuse a value that would make the Builder misbehave, and it
// must not look at the Builder values at all while the feature is off, so an
// untouched deploy can never fail to start because of a setting it never set.
func Test_Validate_Builder(t *testing.T) {
	good := func() {
		SetListenDefaultsForTest()
		globalConfig.AIBuilderEnabled = true
		globalConfig.AIBuilderModel = "gemini-3.8-flash"
		globalConfig.AIBuilderReasoningEffort = "none"
		globalConfig.AIBuilderMaxOutputTokens = 4096
		globalConfig.AIBuilderDailyLimit = 200
		globalConfig.AIBuilderMaxConcurrent = 3
		globalConfig.AIBuilderLLMTimeoutSeconds = 40
	}

	tests := []struct {
		name        string
		mutate      func()
		expectError bool
		expectText  string
	}{
		{"enabled with the shipped values passes", func() {}, false, ""},
		{"a zero LLM timeout is rejected: it means no deadline and would let a stuck call hold an RPC worker and feed the circuit breaker",
			func() { globalConfig.AIBuilderLLMTimeoutSeconds = 0 }, true, "ai_builder_llm_timeout_seconds"},
		{"a negative LLM timeout is rejected", func() { globalConfig.AIBuilderLLMTimeoutSeconds = -1 }, true, "ai_builder_llm_timeout_seconds"},
		{"an LLM timeout at or above the 55 second RPC timeout is rejected: api-manager would give up first while ai-manager kept working and counting",
			func() { globalConfig.AIBuilderLLMTimeoutSeconds = 55 }, true, "ai_builder_llm_timeout_seconds"},
		{"an LLM timeout just below the RPC timeout passes", func() { globalConfig.AIBuilderLLMTimeoutSeconds = 54 }, false, ""},
		{"a zero concurrency is rejected: every call would be BUSY", func() { globalConfig.AIBuilderMaxConcurrent = 0 }, true, "ai_builder_max_concurrent"},
		{"a zero daily limit is rejected: every call would be over the limit", func() { globalConfig.AIBuilderDailyLimit = 0 }, true, "ai_builder_daily_limit"},
		{"a zero max output tokens is rejected", func() { globalConfig.AIBuilderMaxOutputTokens = 0 }, true, "ai_builder_max_output_tokens"},
		{"an empty model is rejected", func() { globalConfig.AIBuilderModel = "" }, true, "ai_builder_model"},
		{"an empty reasoning effort is allowed: it omits the field",
			func() { globalConfig.AIBuilderReasoningEffort = "" }, false, ""},
		{"every offending value is named in one error",
			func() {
				globalConfig.AIBuilderLLMTimeoutSeconds = 0
				globalConfig.AIBuilderMaxConcurrent = 0
				globalConfig.AIBuilderDailyLimit = -3
			},
			true, "ai_builder_daily_limit"},
		{"a broken value is ignored while the feature is off",
			func() {
				globalConfig.AIBuilderEnabled = false
				globalConfig.AIBuilderLLMTimeoutSeconds = 0
				globalConfig.AIBuilderMaxConcurrent = 0
				globalConfig.AIBuilderDailyLimit = 0
				globalConfig.AIBuilderMaxOutputTokens = 0
				globalConfig.AIBuilderModel = ""
			}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			good()
			tt.mutate()

			err := Validate()
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				if tt.expectText != "" && !strings.Contains(err.Error(), tt.expectText) {
					t.Errorf("the error must name %q, got: %v", tt.expectText, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_Validate_Builder_namesEveryOffendingValue(t *testing.T) {
	SetListenDefaultsForTest()
	globalConfig.AIBuilderEnabled = true
	globalConfig.AIBuilderModel = "gemini-3.8-flash"
	globalConfig.AIBuilderMaxOutputTokens = 4096
	globalConfig.AIBuilderLLMTimeoutSeconds = 0
	globalConfig.AIBuilderMaxConcurrent = 0
	globalConfig.AIBuilderDailyLimit = -3

	err := Validate()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"ai_builder_llm_timeout_seconds", "ai_builder_max_concurrent", "ai_builder_daily_limit"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error must name %q, got: %v", want, err)
		}
	}
}

// The whole path a deploy takes: an environment variable, through viper, into
// the Config field the Builder reads. A typo in the env name or a field loaded
// from the wrong key would otherwise pass every test that only checks viper.
func Test_BuilderConfig_envToConfigField(t *testing.T) {
	viper.Reset()
	cmd := &cobra.Command{}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	t.Setenv("AI_BUILDER_ENABLED", "true")
	t.Setenv("AI_BUILDER_MODEL", "model-x")
	t.Setenv("AI_BUILDER_REASONING_EFFORT", "low")
	t.Setenv("AI_BUILDER_MAX_OUTPUT_TOKENS", "1111")
	t.Setenv("AI_BUILDER_DAILY_LIMIT", "22")
	t.Setenv("AI_BUILDER_MAX_CONCURRENT", "33")
	t.Setenv("AI_BUILDER_LLM_TIMEOUT_SECONDS", "44")

	once = sync.Once{}
	LoadGlobalConfig()
	cfg := Get()

	if !cfg.AIBuilderEnabled {
		t.Error("AIBuilderEnabled not loaded from AI_BUILDER_ENABLED")
	}
	if cfg.AIBuilderModel != "model-x" {
		t.Errorf("AIBuilderModel: got %q", cfg.AIBuilderModel)
	}
	if cfg.AIBuilderReasoningEffort != "low" {
		t.Errorf("AIBuilderReasoningEffort: got %q", cfg.AIBuilderReasoningEffort)
	}
	// Each numeric field gets a DIFFERENT value, so a field loaded from the
	// wrong key shows up as a wrong number.
	if cfg.AIBuilderMaxOutputTokens != 1111 {
		t.Errorf("AIBuilderMaxOutputTokens: got %d, want 1111", cfg.AIBuilderMaxOutputTokens)
	}
	if cfg.AIBuilderDailyLimit != 22 {
		t.Errorf("AIBuilderDailyLimit: got %d, want 22", cfg.AIBuilderDailyLimit)
	}
	if cfg.AIBuilderMaxConcurrent != 33 {
		t.Errorf("AIBuilderMaxConcurrent: got %d, want 33", cfg.AIBuilderMaxConcurrent)
	}
	if cfg.AIBuilderLLMTimeoutSeconds != 44 {
		t.Errorf("AIBuilderLLMTimeoutSeconds: got %d, want 44", cfg.AIBuilderLLMTimeoutSeconds)
	}
}
