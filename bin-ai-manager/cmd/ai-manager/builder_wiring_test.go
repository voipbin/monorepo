package main

import (
	"testing"
	"time"

	"monorepo/bin-ai-manager/internal/config"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/openai/"

func Test_builderSettings_keyConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{"gemini url with a google key", config.Config{AnalysisEngineBaseURL: geminiURL, GoogleAPIKey: "g"}, true},
		{"gemini url with only an openai key", config.Config{AnalysisEngineBaseURL: geminiURL, EngineKeyChatGPT: "o"}, false},
		{"gemini url with no key", config.Config{AnalysisEngineBaseURL: geminiURL}, false},
		{"openai url with an openai key", config.Config{AnalysisEngineBaseURL: "https://api.openai.com/v1", EngineKeyChatGPT: "o"}, true},
		{"openai url with only a google key", config.Config{AnalysisEngineBaseURL: "https://api.openai.com/v1", GoogleAPIKey: "g"}, false},
		{"openai url with no key", config.Config{AnalysisEngineBaseURL: "https://api.openai.com/v1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, opts := builderSettings(&tt.cfg)
			if opts.KeyConfigured != tt.want {
				t.Errorf("KeyConfigured: got %v, want %v", opts.KeyConfigured, tt.want)
			}
		})
	}
}

// Every setting reaches the struct that uses it, each with a different value so
// a field wired to the wrong source shows up.
func Test_builderSettings_mapsEverySetting(t *testing.T) {
	cfg := config.Config{
		AnalysisEngineBaseURL:      geminiURL,
		GoogleAPIKey:               "g",
		AIBuilderEnabled:           true,
		AIBuilderModel:             "model-x",
		AIBuilderReasoningEffort:   "low",
		AIBuilderMaxOutputTokens:   1111,
		AIBuilderDailyLimit:        22,
		AIBuilderMaxConcurrent:     33,
		AIBuilderLLMTimeoutSeconds: 44,
	}
	bc, opts := builderSettings(&cfg)

	if bc.Model != "model-x" || bc.ReasoningEffort != "low" || bc.MaxOutputTokens != 1111 {
		t.Errorf("config: %+v", bc)
	}
	if bc.LLMTimeout != 44*time.Second {
		t.Errorf("LLMTimeout: got %v, want 44s (the value is in seconds)", bc.LLMTimeout)
	}
	if !opts.Enabled || opts.DailyLimit != 22 || opts.MaxConcurrent != 33 {
		t.Errorf("options: %+v", opts)
	}
	if bc.SystemPrompt == "" {
		t.Error("the production system prompt must be kept")
	}
}

// The off-by-default promise: an untouched deploy gets a disabled Builder.
func Test_builderSettings_disabledByDefault(t *testing.T) {
	_, opts := builderSettings(&config.Config{AnalysisEngineBaseURL: geminiURL, GoogleAPIKey: "g"})
	if opts.Enabled {
		t.Error("the Builder must be off unless switched on")
	}
}

func Test_builderStartupWarnings(t *testing.T) {
	const openaiURL = "https://api.openai.com/v1"
	tests := []struct {
		name string
		cfg  config.Config
		want int
	}{
		{"off: nothing, even when everything is wrong", config.Config{AnalysisEngineBaseURL: openaiURL, AIBuilderModel: "gemini-3.8-flash"}, 0},
		{"on, key present, matching gemini", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: geminiURL, GoogleAPIKey: "g", AIBuilderModel: "gemini-3.8-flash"}, 0},
		{"on, key present, matching openai", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: openaiURL, EngineKeyChatGPT: "o", AIBuilderModel: "gpt-5"}, 0},
		{"on, no key", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gemini-3.8-flash"}, 1},
		{"on, gemini model on an openai url (rollback)", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: openaiURL, EngineKeyChatGPT: "o", AIBuilderModel: "gemini-3.8-flash"}, 1},
		{"on, openai model on a gemini url", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: geminiURL, GoogleAPIKey: "g", AIBuilderModel: "gpt-5"}, 1},
		{"on, no key and a mismatch: both reported", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gpt-5"}, 2},
		{"model prefix is case-insensitive", config.Config{AIBuilderEnabled: true, AnalysisEngineBaseURL: geminiURL, GoogleAPIKey: "g", AIBuilderModel: "Gemini-3.8-Flash"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, opts := builderSettings(&tt.cfg)
			if got := builderStartupWarnings(&tt.cfg, opts); len(got) != tt.want {
				t.Errorf("got %d warnings %v, want %d", len(got), got, tt.want)
			}
		})
	}
}
