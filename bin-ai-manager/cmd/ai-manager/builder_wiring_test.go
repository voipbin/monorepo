package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/confighandler/storetest"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/openai/"

// testKeyFunc builds the key supplier of the analysis engine the way run() does, on a store with
// no rows, so the value comes from the environment fields of cfg.
func testKeyFunc(t *testing.T, cfg *config.Config) aiconfig.KeyFunc {
	t.Helper()

	h, err := aiconfig.New(cfg, storetest.NewMemory())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return newPlatformKeys(h, cfg).analysis
}

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
			_, opts := builderSettings(&tt.cfg, testKeyFunc(t, &tt.cfg))
			got, err := opts.KeyConfigured(context.Background())
			if err != nil || got != tt.want {
				t.Errorf("KeyConfigured: got %v %v, want %v", got, err, tt.want)
			}
		})
	}
}

// The check reads the key on every call: a key that appears later is seen by the next check.
func Test_builderSettings_keyConfiguredIsReadEveryTime(t *testing.T) {
	calls := 0
	key := ""
	f := func(context.Context) (string, error) {
		calls++
		return key, nil
	}
	_, opts := builderSettings(&config.Config{}, f)

	if got, _ := opts.KeyConfigured(context.Background()); got {
		t.Errorf("Wrong match. expect: not configured")
	}
	key = "now-set"
	if got, _ := opts.KeyConfigured(context.Background()); !got {
		t.Errorf("Wrong match. expect: configured on the next call")
	}
	if calls != 2 {
		t.Errorf("Wrong match. expect: 2 reads, got: %d", calls)
	}
}

// A key that can not be read is an error, not "not configured".
func Test_builderSettings_keyConfiguredReadError(t *testing.T) {
	boom := errors.New("read failed")
	_, opts := builderSettings(&config.Config{}, func(context.Context) (string, error) { return "", boom })

	got, err := opts.KeyConfigured(context.Background())
	if got || !errors.Is(err, boom) {
		t.Errorf("Wrong match. expect: false and the read error, got: %v %v", got, err)
	}
}

// Every setting reaches the struct that uses it, each with a different value so
// a field wired to the wrong source shows up.
func Test_builderSettings_mapsEverySetting(t *testing.T) {
	cfg := config.Config{
		AnalysisEngineBaseURL:      geminiURL,
		GoogleAPIKey:               "g",
		AIBuilderModel:             "model-x",
		AIBuilderReasoningEffort:   "low",
		AIBuilderMaxOutputTokens:   1111,
		AIBuilderDailyLimit:        22,
		AIBuilderMaxConcurrent:     33,
		AIBuilderLLMTimeoutSeconds: 44,
	}
	bc, opts := builderSettings(&cfg, testKeyFunc(t, &cfg))

	if bc.Model != "model-x" || bc.ReasoningEffort != "low" || bc.MaxOutputTokens != 1111 {
		t.Errorf("config: %+v", bc)
	}
	if bc.LLMTimeout != 44*time.Second {
		t.Errorf("LLMTimeout: got %v, want 44s (the value is in seconds)", bc.LLMTimeout)
	}
	if opts.DailyLimit != 22 || opts.MaxConcurrent != 33 {
		t.Errorf("options: %+v", opts)
	}
	if bc.SystemPrompt == "" {
		t.Error("the production system prompt must be kept")
	}
}

func Test_builderStartupNotes(t *testing.T) {
	const openaiURL = "https://api.openai.com/v1"
	tests := []struct {
		name       string
		cfg        config.Config
		configured bool
		errKey     error
		wantInfos  int
		wantWarns  int
	}{
		{"key present, matching gemini", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gemini-3.8-flash"}, true, nil, 0, 0},
		{"key present, matching openai", config.Config{AnalysisEngineBaseURL: openaiURL, AIBuilderModel: "gpt-5"}, true, nil, 0, 0},
		{"no key", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gemini-3.8-flash"}, false, nil, 1, 0},
		{"gemini model on an openai url (rollback)", config.Config{AnalysisEngineBaseURL: openaiURL, AIBuilderModel: "gemini-3.8-flash"}, true, nil, 0, 1},
		{"openai model on a gemini url", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gpt-5"}, true, nil, 0, 1},
		{"no key and a mismatch: only the missing key is reported", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gpt-5"}, false, nil, 1, 0},
		{"model prefix is case-insensitive", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "Gemini-3.8-Flash"}, true, nil, 0, 0},
		{"a key that can not be read is a warning", config.Config{AnalysisEngineBaseURL: geminiURL, AIBuilderModel: "gemini-3.8-flash"}, false, errors.New("read failed"), 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infos, warns := builderStartupNotes(&tt.cfg, tt.configured, tt.errKey)
			if len(infos) != tt.wantInfos || len(warns) != tt.wantWarns {
				t.Errorf("got infos %v and warnings %v, want %d and %d", infos, warns, tt.wantInfos, tt.wantWarns)
			}
		})
	}
}
