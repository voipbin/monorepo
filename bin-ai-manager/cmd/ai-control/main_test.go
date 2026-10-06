package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
)

type fakeAIGetter struct {
	res   *ai.AI
	err   error
	calls int
}

func (f *fakeAIGetter) Get(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	f.calls++
	return f.res, f.err
}

func Test_validateEngineModelCreate(t *testing.T) {
	tests := []struct {
		name      string
		model     ai.EngineModel
		key       string
		wantError bool
		wantMsg   string
		hideInput string // must not appear in the error message
	}{
		{"empty_skips_validation", "", "", false, "", ""},
		{"catalog_model_passes", "anthropic.claude-haiku-4.5", "", false, "", ""},
		{"direct_passthrough_passes", "openai.gpt-4o", "", false, "", ""},
		{"not_in_catalog_fails", "anthropic.claude-opus-4", "", true, "invalid engine model: anthropic.claude-opus-4", ""},
		{"custom_with_key_passes", "openrouter.vendor/model-a", "dummy-new-key", false, "", ""},
		{"custom_empty_key_fails", "openrouter.vendor/model-a", "", true, "an API key is required for custom OpenRouter models", ""},
		{"custom_blank_key_fails", "openrouter.vendor/model-a", "  ", true, "an API key is required for custom OpenRouter models", ""},
		{"custom_bad_id_fails_without_echo", "openrouter.dummy-key-not-real", "dummy-new-key", true, "invalid engine model: the OpenRouter model ID is not valid", "dummy-key-not-real"},
		{"custom_upper_prefix_bad_id_fails_without_echo", "OpenRouter.dummy-key-not-real", "dummy-new-key", true, "invalid engine model: the OpenRouter model ID is not valid", "dummy-key-not-real"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEngineModelCreate(tt.model, tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && err.Error() != tt.wantMsg {
				t.Errorf("message. expect: %q, got: %q", tt.wantMsg, err.Error())
			}
			if err != nil && tt.hideInput != "" && strings.Contains(err.Error(), tt.hideInput) {
				t.Errorf("message echoes the input: %v", err)
			}
		})
	}
}

func Test_validateEngineModelUpdate(t *testing.T) {
	tests := []struct {
		name        string
		stored      ai.EngineModel
		getErr      error
		requested   ai.EngineModel
		key         string
		wantError   bool
		expectFetch bool
	}{
		{"empty_skips_validation_and_fetch", "openai.gpt-5", nil, "", "", false, false},
		{"empty_model_with_empty_key_skips", "openrouter.vendor/model-a", nil, "", "", false, false},
		{"unchanged_legacy_passes", "anthropic.claude-opus-4", nil, "anthropic.claude-opus-4", "", false, true},
		{"changed_to_invalid_fails", "openai.gpt-5", nil, "anthropic.claude-opus-4", "", true, true},
		{"changed_to_catalog_passes", "anthropic.claude-opus-4", nil, "anthropic.claude-haiku-4.5", "", false, true},
		{"fetch_error_fails", "", errors.New("not found"), "openai.gpt-5", "", true, true},
		{"changed_to_custom_with_key_passes", "openai.gpt-5", nil, "openrouter.vendor/model-a", "dummy-new-key", false, true},
		{"changed_to_custom_empty_key_fails", "openai.gpt-5", nil, "openrouter.vendor/model-a", "", true, true},
		{"unchanged_custom_empty_key_fails", "openrouter.vendor/model-a", nil, "openrouter.vendor/model-a", "", true, true},
		{"unchanged_custom_with_key_passes", "openrouter.vendor/model-a", nil, "openrouter.vendor/model-a", "dummy-new-key", false, true},
		{"changed_to_bad_custom_fails", "openai.gpt-5", nil, "openrouter.dummy-key-not-real", "dummy-new-key", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &fakeAIGetter{res: &ai.AI{EngineModel: tt.stored}, err: tt.getErr}
			err := validateEngineModelUpdate(context.Background(), g, uuid.Must(uuid.NewV4()), tt.requested, tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && strings.Contains(err.Error(), "dummy-key-not-real") {
				t.Errorf("message echoes the input: %v", err)
			}
			if (g.calls > 0) != tt.expectFetch {
				t.Errorf("fetch calls = %d, expectFetch %v", g.calls, tt.expectFetch)
			}
		})
	}
}
