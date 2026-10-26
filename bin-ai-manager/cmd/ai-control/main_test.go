package main

import (
	"context"
	"errors"
	"io"
	"os"
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
		{"custom_leading_space_bad_id_fails_without_echo", " openrouter.dummy-key-not-real", "dummy-new-key", true, "invalid engine model: the OpenRouter model ID is not valid", "dummy-key-not-real"},
		{"custom_leading_tab_bad_id_fails_without_echo", "\topenrouter.dummy-key-not-real", "dummy-new-key", true, "invalid engine model: the OpenRouter model ID is not valid", "dummy-key-not-real"},
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
			err := validateEngineModelUpdate(context.Background(), g, uuid.Must(uuid.NewV4()), tt.requested, keyArg(tt.key))
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

func Test_withoutEndpointCiphertext(t *testing.T) {
	in := &ai.AI{EngineBaseURL: "https://llm.example.com", EngineBaseKeyEnc: []byte("cipher"), EngineBaseKey: "plain"}

	out, ok := withoutEndpointCiphertext(in).(*ai.AI)
	if !ok || out.EngineBaseKeyEnc != nil || out.EngineBaseKey != "" || out.EngineBaseURL != in.EngineBaseURL {
		t.Errorf("the printed AI must carry no key material and keep the url. got: %+v", out)
	}
	if in.EngineBaseKeyEnc == nil || in.EngineBaseKey == "" {
		t.Errorf("the input must stay untouched")
	}

	list, ok := withoutEndpointCiphertext([]*ai.AI{in, in}).([]*ai.AI)
	if !ok || len(list) != 2 || list[0].EngineBaseKeyEnc != nil || list[1].EngineBaseKeyEnc != nil {
		t.Errorf("a list must be cleared too")
	}
}

func Test_engineCheckError_CustomEndpointPrefix(t *testing.T) {
	err := engineCheckError(ai.EngineModel("customendpoint.bad id with sentinel-secret"), ai.ErrInvalidEngineModel)
	if err == nil || strings.Contains(err.Error(), "sentinel-secret") {
		t.Errorf("the rejected value must not be echoed. got: %v", err)
	}
}

// printJSON is what the operator sees: no ciphertext, whatever the handler returned.
func Test_printJSON_HidesEndpointCiphertext(t *testing.T) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	errPrint := printJSON(&ai.AI{EngineBaseURL: "https://llm.example.com", EngineBaseKeyEnc: []byte("cipher-sentinel"), EngineBaseKey: "plain-sentinel"})
	_ = w.Close()
	os.Stdout = orig

	out, _ := io.ReadAll(r)
	if errPrint != nil {
		t.Fatalf("unexpected error: %v", errPrint)
	}
	if strings.Contains(string(out), "Y2lwaGVyLXNlbnRpbmVs") || strings.Contains(string(out), "cipher-sentinel") || strings.Contains(string(out), "plain-sentinel") {
		t.Errorf("the output carries key material: %s", out)
	}
	if !strings.Contains(string(out), "https://llm.example.com") {
		t.Errorf("the url must be printed: %s", out)
	}
}

// keyArg maps the table's empty key to an omitted flag, like the update command does.
func keyArg(s string) *string { return engineKeyFlag(s) }

func Test_engineKeyFlag(t *testing.T) {
	if got := engineKeyFlag(""); got != nil {
		t.Errorf("an empty flag must be an omitted key, got: %q", *got)
	}
	if got := engineKeyFlag("dummy-key"); got == nil || *got != "dummy-key" {
		t.Errorf("a value must be passed as is, got: %v", got)
	}
}

func Test_validateEngineModelUpdate_mergesStoredKey(t *testing.T) {
	tests := []struct {
		name      string
		stored    ai.EngineModel
		storedKey string
		requested ai.EngineModel
		key       *string
		wantError bool
	}{
		{"omitted_key_uses_stored_key_for_custom_model", "openrouter.vendor/model-a", "dummy-stored-key", "openrouter.vendor/model-a", nil, false},
		{"omitted_key_without_stored_key_fails", "openrouter.vendor/model-a", "", "openrouter.vendor/model-a", nil, true},
		{"vendor_change_drops_stored_key", "openai.gpt-5", "dummy-stored-key", "openrouter.vendor/model-a", nil, true},
		{"explicit_value_wins", "openrouter.vendor/model-a", "", "openrouter.vendor/model-a", keyArg("dummy-new-key"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &fakeAIGetter{res: &ai.AI{EngineModel: tt.stored, EngineKey: tt.storedKey}}
			err := validateEngineModelUpdate(context.Background(), g, uuid.Must(uuid.NewV4()), tt.requested, tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
