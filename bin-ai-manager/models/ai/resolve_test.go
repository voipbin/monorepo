package ai

import "testing"

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
		{"raw_openrouter", "openrouter.meta-llama/llama-3-70b", OutcomeRejected, "", "", false},
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
