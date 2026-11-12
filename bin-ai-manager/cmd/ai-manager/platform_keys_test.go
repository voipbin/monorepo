package main

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler/storetest"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	testGeminiURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
	testOpenAIURL = "https://api.openai.com/v1"
)

func testMasterKeyList() string {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(0x30 + i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(raw)
}

// Each consumer gets the key of its provider, and a changed key is used by the next call of every
// consumer. The summary and the analysis have a provider of their own, which is not the one of
// the conversation (always the OpenAI default endpoint) or of the audit and proposal (always
// Gemini).
func Test_newPlatformKeys(t *testing.T) {
	tests := []struct {
		name            string
		summaryURL      string
		analysisURL     string
		expectSummary   string
		expectAnalysis  string
		storedOpenAIKey string
		storedGemini    string
	}{
		{"summary gemini, analysis openai", testGeminiURL, testOpenAIURL, "gemini-value", "openai-value", "openai-value", "gemini-value"},
		{"summary openai, analysis gemini", testOpenAIURL, testGeminiURL, "openai-value", "gemini-value", "openai-value", "gemini-value"},
		{"both default", "", "", "openai-value", "openai-value", "openai-value", "gemini-value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ConfigEncryptionKeys:  testMasterKeyList(),
				SummaryEngineBaseURL:  tt.summaryURL,
				AnalysisEngineBaseURL: tt.analysisURL,
			}
			h, err := aiconfig.New(cfg, storetest.NewMemory())
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			actor := uuid.Must(uuid.NewV4())
			if _, err := h.SetSecret(context.Background(), actor, configkeys.KeyOpenAICompatAPIKey, tt.storedOpenAIKey); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if _, err := h.SetSecret(context.Background(), actor, configkeys.KeyGeminiAPIKey, tt.storedGemini); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			keys := newPlatformKeys(h, cfg)
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

			check("stored", "openai-value", tt.expectSummary, tt.expectAnalysis, "gemini-value")

			// a rotation is used by the next call of every consumer, with the same functions
			if _, err := h.SetSecret(context.Background(), actor, configkeys.KeyOpenAICompatAPIKey, "openai-rotated"); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if _, err := h.SetSecret(context.Background(), actor, configkeys.KeyGeminiAPIKey, "gemini-rotated"); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			rot := func(stored string) string {
				if stored == "gemini-value" {
					return "gemini-rotated"
				}
				return "openai-rotated"
			}
			check("rotated", "openai-rotated", rot(tt.expectSummary), rot(tt.expectAnalysis), "gemini-rotated")
		})
	}
}
