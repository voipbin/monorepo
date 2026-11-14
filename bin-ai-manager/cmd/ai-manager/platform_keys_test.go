package main

import (
	"context"
	"testing"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	testGeminiURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
	testOpenAIURL = "https://api.openai.com/v1"
)

// Each consumer gets the key of its provider, and a changed variable is used by the next call of
// every consumer. The summary and the analysis have a provider of their own, which is not the one
// of the conversation (always the OpenAI default endpoint) or of the audit and proposal (always
// Gemini).
func Test_newPlatformKeys(t *testing.T) {
	tests := []struct {
		name           string
		summaryURL     string
		analysisURL    string
		expectSummary  string
		expectAnalysis string
	}{
		{"summary gemini, analysis openai", testGeminiURL, testOpenAIURL, "gemini-value", "openai-value"},
		{"summary openai, analysis gemini", testOpenAIURL, testGeminiURL, "openai-value", "gemini-value"},
		{"both default", "", "", "openai-value", "openai-value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvOpenAIAPIKey, "openai-value")
			t.Setenv(configkeys.EnvGoogleAPIKey, "gemini-value")

			cfg := &config.Config{
				SummaryEngineBaseURL:  tt.summaryURL,
				AnalysisEngineBaseURL: tt.analysisURL,
			}

			keys := newPlatformKeys(cfg)
			check := func(round string, conv, summary, analysis, gemini string) {
				for _, c := range []struct {
					name   string
					fn     aiconfig.KeyFunc
					expect string
				}{
					{"conversation", keys.conversation, conv},
					{"summary", keys.summary, summary},
					{"analysis", keys.analysis, analysis},
					{"gemini", keys.gemini, gemini},
				} {
					if got, err := c.fn(context.Background()); err != nil || got != c.expect {
						t.Errorf("Wrong match. %s %s expect: %q, got: %q %v", round, c.name, c.expect, got, err)
					}
				}
			}

			check("set", "openai-value", tt.expectSummary, tt.expectAnalysis, "gemini-value")

			// a rotation is used by the next call of every consumer, with the same functions
			t.Setenv(configkeys.EnvOpenAIAPIKey, "openai-rotated")
			t.Setenv(configkeys.EnvGoogleAPIKey, "gemini-rotated")
			rot := func(v string) string {
				if v == "gemini-value" {
					return "gemini-rotated"
				}
				return "openai-rotated"
			}
			check("rotated", "openai-rotated", rot(tt.expectSummary), rot(tt.expectAnalysis), "gemini-rotated")
		})
	}
}

// A variable that is not set or still a placeholder is not configured for every consumer.
func Test_newPlatformKeys_notConfigured(t *testing.T) {
	t.Setenv(configkeys.EnvOpenAIAPIKey, "")
	t.Setenv(configkeys.EnvGoogleAPIKey, "[[BIN_MANAGER__GOOGLE_API_KEY]]")

	keys := newPlatformKeys(&config.Config{SummaryEngineBaseURL: testGeminiURL})
	for name, fn := range map[string]aiconfig.KeyFunc{"conversation": keys.conversation, "summary": keys.summary, "analysis": keys.analysis, "gemini": keys.gemini} {
		if got, err := fn(context.Background()); err != nil || got != "" {
			t.Errorf("Wrong match. %s expect: empty, got: %q %v", name, got, err)
		}
	}
}

// The suppliers read the environment of the process and nothing else, whatever the variables hold.
func Test_platformKeys_readTheEnvironment(t *testing.T) {
	t.Setenv(configkeys.EnvOpenAIAPIKey, "openai-value")
	t.Setenv(configkeys.EnvGoogleAPIKey, "")

	keys := newPlatformKeys(&config.Config{})
	if got, errKey := keys.conversation(context.Background()); errKey != nil || got != "openai-value" {
		t.Errorf("Wrong match. expect: openai-value, got: %q %v", got, errKey)
	}
	if got, errKey := keys.gemini(context.Background()); errKey != nil || got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q %v", got, errKey)
	}
	if _, _, errPair := aiconfig.MCPOAuthClient(configkeys.VendorGitHub); errPair != nil {
		t.Errorf("Wrong match. expect: no error, got: %v", errPair)
	}
}
