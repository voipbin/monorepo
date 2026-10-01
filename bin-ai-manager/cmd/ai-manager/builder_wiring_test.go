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
