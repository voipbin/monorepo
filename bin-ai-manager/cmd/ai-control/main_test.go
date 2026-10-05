package main

import (
	"context"
	"errors"
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
		wantError bool
	}{
		{"empty_skips_validation", "", false},
		{"catalog_model_passes", "anthropic.claude-haiku-4.5", false},
		{"direct_passthrough_passes", "openai.gpt-4o", false},
		{"not_in_catalog_fails", "anthropic.claude-opus-4", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEngineModelCreate(tt.model); (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
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
		wantError   bool
		expectFetch bool
	}{
		{"empty_skips_validation_and_fetch", "openai.gpt-5", nil, "", false, false},
		{"unchanged_legacy_passes", "anthropic.claude-opus-4", nil, "anthropic.claude-opus-4", false, true},
		{"changed_to_invalid_fails", "openai.gpt-5", nil, "anthropic.claude-opus-4", true, true},
		{"changed_to_catalog_passes", "anthropic.claude-opus-4", nil, "anthropic.claude-haiku-4.5", false, true},
		{"fetch_error_fails", "", errors.New("not found"), "openai.gpt-5", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &fakeAIGetter{res: &ai.AI{EngineModel: tt.stored}, err: tt.getErr}
			err := validateEngineModelUpdate(context.Background(), g, uuid.Must(uuid.NewV4()), tt.requested)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
			}
			if (g.calls > 0) != tt.expectFetch {
				t.Errorf("fetch calls = %d, expectFetch %v", g.calls, tt.expectFetch)
			}
		})
	}
}
