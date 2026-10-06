package ai

import (
	"errors"
	"strings"
	"testing"
)

func Test_ExistingElevenModelsAreDirectCatalogEntries(t *testing.T) {
	for _, id := range []EngineModel{
		EngineModelGeminiGemini2Dot5Flash, EngineModelGeminiGemini2Dot5Pro, EngineModelGeminiGemini2Dot0Flash, EngineModelGeminiGeminiProLatest,
		EngineModelOpenaiGPT5Dot2, EngineModelOpenaiGPT5Dot1, EngineModelOpenaiGPT5, EngineModelOpenaiGPT5Mini, EngineModelOpenaiGPT5Nano,
		EngineModelGrok3, EngineModelGrok3Mini,
	} {
		r, o := ResolveEngine(id)
		if o != OutcomeCatalog || r.Entry == nil || r.Entry.Route != RouteDirect {
			t.Errorf("%s must stay a direct catalog entry", id)
		}
		if r.RunnerType != string(id) || r.BlankKey {
			t.Errorf("%s: direct entry must keep its type and key", id)
		}
	}
}

func Test_ResolveEngine(t *testing.T) {
	tests := []struct {
		name           string
		model          EngineModel
		expectOutcome  Outcome
		expectRoute    Route
		expectRunner   string
		expectBlankKey bool
	}{
		{"catalog_direct", "openai.gpt-5", OutcomeCatalog, RouteDirect, "openai.gpt-5", false},
		{"catalog_openrouter", "anthropic.claude-haiku-4.5", OutcomeCatalog, RouteOpenRouter, "platform_openrouter.anthropic/claude-haiku-4.5", true},
		{"catalog_openrouter_meta", "meta.llama-3.3-70b-instruct", OutcomeCatalog, RouteOpenRouter, "platform_openrouter.meta-llama/llama-3.3-70b-instruct", true},
		{"direct_passthrough_openai", "openai.gpt-4o", OutcomeDirectPassthrough, "", "openai.gpt-4o", false},
		{"direct_passthrough_gemini", "gemini.gemini-1.5-pro", OutcomeDirectPassthrough, "", "gemini.gemini-1.5-pro", false},
		{"direct_passthrough_grok", "grok.grok-4", OutcomeDirectPassthrough, "", "grok.grok-4", false},
		{"not_in_catalog_anthropic", "anthropic.claude-opus-4", OutcomeRejected, "", "", false},
		{"custom_openrouter", "openrouter.meta-llama/llama-3-70b", OutcomeCustomOpenRouter, "", "openrouter.meta-llama/llama-3-70b", false},
		{"custom_openrouter_no_slash", "openrouter.x", OutcomeRejected, "", "", false},
		{"custom_openrouter_variant", "openrouter.vendor/model:free", OutcomeRejected, "", "", false},
		{"custom_openrouter_router", "openrouter.openrouter/auto", OutcomeRejected, "", "", false},
		{"custom_openrouter_upper_prefix", "OpenRouter.vendor/model", OutcomeRejected, "", "", false},
		{"custom_placeholder_id", "custom.openrouter", OutcomeRejected, "", "", false},
		{"internal_type", "platform_openrouter.x", OutcomeRejected, "", "", false},
		{"empty", "", OutcomeRejected, "", "", false},
		{"empty_name", "openai.", OutcomeRejected, "", "", false},
		{"unknown_prefix", "unknown.x", OutcomeRejected, "", "", false},
		{"no_dot", "invalid", OutcomeRejected, "", "", false},
		{"case_variant", "OpenAI.gpt-4o", OutcomeRejected, "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, o := ResolveEngine(tt.model)
			if o != tt.expectOutcome {
				t.Fatalf("outcome. expect: %v, got: %v", tt.expectOutcome, o)
			}
			if r.RunnerType != tt.expectRunner {
				t.Errorf("runner type. expect: %q, got: %q", tt.expectRunner, r.RunnerType)
			}
			if r.BlankKey != tt.expectBlankKey {
				t.Errorf("blank key. expect: %v, got: %v", tt.expectBlankKey, r.BlankKey)
			}
			if tt.expectOutcome == OutcomeCatalog && r.Entry.Route != tt.expectRoute {
				t.Errorf("route. expect: %v, got: %v", tt.expectRoute, r.Entry.Route)
			}
			if tt.expectOutcome != OutcomeCatalog && r.Entry != nil {
				t.Errorf("entry must be nil for non-catalog outcomes")
			}
		})
	}
}

func Test_ResolveEngineAllOpenRouterEntriesBlankKey(t *testing.T) {
	for _, e := range catalog {
		if e.Route == RouteCustomOpenRouter {
			// The custom entry is a placeholder, never a selectable model.
			if _, o := ResolveEngine(e.ID); o != OutcomeRejected {
				t.Errorf("%s must be rejected as a stored value. got: %v", e.ID, o)
			}
			continue
		}

		r, o := ResolveEngine(e.ID)
		if o != OutcomeCatalog {
			t.Errorf("%s must resolve", e.ID)
			continue
		}
		if (e.Route == RouteOpenRouter) != r.BlankKey {
			t.Errorf("%s: blank key must match openrouter route", e.ID)
		}
	}
}

func Test_ValidateCustomModelID(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		expect bool
	}{
		{"plain", "mistralai/mistral-large-2411", true},
		{"dots_and_dashes", "meta-llama/llama-3.3-70b-instruct", true},
		{"dot_in_slug", "mistralai/mistral-medium-3.1", true},
		{"case_is_kept", "Mistralai/Large", true},
		{"max_length", strings.Repeat("a", 121) + "/" + strings.Repeat("b", 122), true},
		{"too_long", strings.Repeat("a", 122) + "/" + strings.Repeat("b", 122), false},
		{"empty", "", false},
		{"space", "vendor/mod el", false},
		{"control_char", "vendor/model\n", false},
		{"comma", "vendor/a,b", false},
		{"tilde", "~vendor/model", false},
		{"at", "vendor/model@x", false},
		{"variant_free", "vendor/model:free", false},
		{"variant_nitro", "vendor/model:nitro", false},
		{"variant_online", "vendor/model:online", false},
		{"variant_thinking", "vendor/model:thinking", false},
		{"variant_floor", "vendor/model:floor", false},
		{"no_slash", "model", false},
		{"two_slashes", "a/b/c", false},
		{"leading_dot", ".vendor/model", false},
		{"leading_dash", "-vendor/model", false},
		{"leading_underscore", "_vendor/model", false},
		{"slug_leading_dot", "vendor/.model", false},
		{"router_lower", "openrouter/auto", false},
		{"router_mixed", "OpenRouter/auto", false},
		{"router_upper", "OPENROUTER/free", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCustomModelID(tt.id); got != tt.expect {
				t.Errorf("Wrong result. id: %q, expect: %v, got: %v", tt.id, tt.expect, got)
			}
		})
	}
}

func Test_ResolveEngineCustomRequiresKey(t *testing.T) {
	r, o := ResolveEngine("openrouter.vendor/model-a")
	if o != OutcomeCustomOpenRouter {
		t.Fatalf("outcome. expect: %v, got: %v", OutcomeCustomOpenRouter, o)
	}
	if !r.RequireKey || r.BlankKey || r.Entry != nil {
		t.Errorf("custom must require a key and never blank it. got: %+v", r)
	}
}

func Test_ResolveEngineCustomNeverPromotes(t *testing.T) {
	tests := []struct {
		name         string
		model        EngineModel
		expectCustom bool
	}{
		{"upper_prefix", "OpenRouter.x/y", false},
		{"upper_platform", "PLATFORM_OPENROUTER.x/y", false},
		{"leading_space", " openrouter.x/y", false},
		{"empty_id", "openrouter.", false},
		{"platform_in_id_without_slash", "openrouter.platform_openrouter.x", false},
		{"platform_in_author", "openrouter.platform_openrouter/x", true},
		{"placeholder", "custom.openrouter", false},
		{"internal_type", "platform_openrouter.x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, o := ResolveEngine(tt.model)
			if (o == OutcomeCustomOpenRouter) != tt.expectCustom {
				t.Fatalf("outcome. model: %q, expect custom: %v, got: %v", tt.model, tt.expectCustom, o)
			}
			if !tt.expectCustom {
				if o != OutcomeRejected {
					t.Errorf("must be rejected. got: %v", o)
				}
				return
			}

			// The runner lowercases the part before the first dot, so it must be exactly "openrouter".
			if first := strings.SplitN(r.RunnerType, ".", 2)[0]; first != "openrouter" {
				t.Errorf("runner type first segment. expect: openrouter, got: %s", first)
			}
			if r.BlankKey {
				t.Errorf("custom must never blank the key")
			}
		})
	}
}

func Test_ValidateEngine(t *testing.T) {
	tests := []struct {
		name         string
		model        EngineModel
		key          string
		modelChanged bool
		expect       error
	}{
		{"custom_changed_with_key", "openrouter.a/b", "dummy-new-key", true, nil},
		{"custom_changed_empty_key", "openrouter.a/b", "", true, ErrEngineKeyRequired},
		{"custom_changed_blank_key", "openrouter.a/b", "   ", true, ErrEngineKeyRequired},
		{"custom_unchanged_with_key", "openrouter.a/b", "dummy-new-key", false, nil},
		{"custom_unchanged_empty_key", "openrouter.a/b", "", false, ErrEngineKeyRequired},
		{"custom_unchanged_blank_key", "openrouter.a/b", "   ", false, ErrEngineKeyRequired},
		{"legacy_unchanged_with_key", "openrouter.x/y", "dummy-new-key", false, nil},
		{"invalid_custom_changed", "openrouter.a", "dummy-new-key", true, ErrInvalidEngineModel},
		{"invalid_custom_changed_empty_key", "openrouter.a", "", true, ErrInvalidEngineModel},
		{"invalid_custom_unchanged_with_key", "openrouter.a", "dummy-new-key", false, nil},
		{"invalid_model_changed", "anthropic.claude-opus-4", "", true, ErrInvalidEngineModel},
		{"legacy_invalid_unchanged", "anthropic.claude-opus-4", "", false, nil},
		{"direct_empty_key", "openai.gpt-5", "", true, nil},
		{"platform_empty_key", "anthropic.claude-haiku-4.5", "", true, nil},
		// swapped arguments: a model string in the key slot must not trigger the model rules
		{"swapped_arguments", "openai.gpt-5", "openrouter.a/b", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateEngine(tt.model, tt.key, tt.modelChanged); !errors.Is(got, tt.expect) {
				t.Errorf("Wrong result. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
