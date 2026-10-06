package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func Test_CatalogInvariants(t *testing.T) {
	seen := map[EngineModel]bool{}
	for _, e := range catalog {
		if seen[e.ID] {
			t.Errorf("duplicate id %s", e.ID)
		}
		seen[e.ID] = true
		if e.Label == "" || e.Vendor == "" {
			t.Errorf("%s: label and vendor are required", e.ID)
		}
		if e.Description == "" {
			t.Errorf("%s: description is required", e.ID)
		}
		if e.Route == RouteOpenRouter {
			if e.UpstreamSlug == "" || strings.Contains(e.UpstreamSlug, ":") {
				t.Errorf("%s: openrouter entry needs a plain slug (no variant suffix): %q", e.ID, e.UpstreamSlug)
			}
		}
		if e.Route == RouteDirect && e.UpstreamSlug != "" {
			t.Errorf("%s: direct entry must not carry a slug", e.ID)
		}
		if e.Route != RouteDirect && e.Route != RouteOpenRouter && e.Route != RouteCustomOpenRouter {
			t.Errorf("%s: unknown route %q", e.ID, e.Route)
		}
		if e.Route == RouteCustomOpenRouter && (e.UpstreamSlug != "" || e.CustomPrefix != "openrouter.") {
			t.Errorf("%s: custom entry needs no slug and the openrouter. prefix: %q, %q", e.ID, e.UpstreamSlug, e.CustomPrefix)
		}
		if e.Route != RouteCustomOpenRouter && e.CustomPrefix != "" {
			t.Errorf("%s: only the custom entry may carry a custom prefix", e.ID)
		}
		if strings.HasPrefix(string(e.ID), "openrouter.") {
			t.Errorf("%s: no catalog id may start with the custom prefix", e.ID)
		}
		for _, tag := range e.Tags {
			if tag != TagLowCost {
				t.Errorf("%s: unknown tag %q", e.ID, tag)
			}
		}
	}

	custom := 0
	for _, e := range catalog {
		if e.Route == RouteCustomOpenRouter {
			custom++
		}
	}
	if custom != 1 {
		t.Errorf("exactly one custom entry is expected. got: %d", custom)
	}
}

// customOpenRouterAllowedFields are the only fields of the custom entry that may mention "openrouter".
var customOpenRouterAllowedFields = map[string]bool{"id": true, "label": true, "vendor": true, "description": true, "model_id_prefix": true}

func Test_CatalogPublicViewHidesInternals(t *testing.T) {
	b, err := json.Marshal(CatalogView())
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatal(err)
	}

	prefixed := 0
	for _, item := range items {
		if _, ok := item["model_id_prefix"]; ok {
			prefixed++
		}

		if item["key_mode"] != KeyModeOwnRequired {
			// Regular entries keep the full banned-token check.
			raw, _ := json.Marshal(item)
			for _, banned := range []string{"slug", "route", "openrouter", "meta-llama/"} {
				if strings.Contains(strings.ToLower(string(raw)), banned) {
					t.Errorf("%v: public catalog leaks %q", item["id"], banned)
				}
			}
			continue
		}

		// The custom entry is checked by key name, because its id, label and prefix
		// legitimately contain "openrouter" (which itself contains "route").
		wantKeys := []string{"id", "label", "vendor", "recommended", "tags", "description", "platform_managed", "key_mode", "model_id_prefix"}
		if len(item) != len(wantKeys) {
			t.Errorf("custom entry keys. expect: %v, got: %v", wantKeys, item)
		}
		for _, k := range wantKeys {
			if _, ok := item[k]; !ok {
				t.Errorf("custom entry misses key %s", k)
			}
		}
		for k, v := range item {
			lower := strings.ToLower(fmt.Sprint(v))
			if strings.Contains(lower, "slug") || strings.Contains(lower, "meta-llama/") {
				t.Errorf("custom entry field %s leaks internals: %v", k, v)
			}
			if !customOpenRouterAllowedFields[k] && strings.Contains(lower, "openrouter") {
				t.Errorf("custom entry field %s must not mention openrouter: %v", k, v)
			}
		}
	}
	if prefixed != 1 {
		t.Errorf("exactly one entry must carry model_id_prefix. got: %d", prefixed)
	}
}

func Test_CatalogKeyModeMatchesRoute(t *testing.T) {
	view := CatalogView()
	b, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	expects := map[Route]string{
		RouteDirect:           KeyModeOwnOrDefault,
		RouteOpenRouter:       KeyModePlatform,
		RouteCustomOpenRouter: KeyModeOwnRequired,
	}
	for i, e := range catalog {
		if view[i].KeyMode != expects[e.Route] {
			t.Errorf("%s: key_mode. expect: %s, got: %s", e.ID, expects[e.Route], view[i].KeyMode)
		}
		if _, ok := decoded[i]["key_mode"]; !ok {
			t.Errorf("%s: key_mode must always be present", e.ID)
		}

		_, hasPrefix := decoded[i]["model_id_prefix"]
		if hasPrefix != (e.Route == RouteCustomOpenRouter) {
			t.Errorf("%s: model_id_prefix presence. route: %s, got: %v", e.ID, e.Route, hasPrefix)
		}
	}
}

func Test_CatalogViewShape(t *testing.T) {
	view := CatalogView()
	if len(view) != len(catalog) {
		t.Fatalf("view length mismatch. expect: %d, got: %d", len(catalog), len(view))
	}
	rec := 0
	for _, v := range view {
		if v.Tags == nil {
			t.Errorf("%s: tags must be a non-nil slice", v.ID)
		}
		if v.Recommended {
			rec++
		}
	}
	if rec != 1 || !hasRecommended(view, EngineModelGeminiGemini2Dot5Flash) {
		t.Errorf("only gemini 2.5 flash must be recommended")
	}
}

func hasRecommended(view []ModelInfo, id EngineModel) bool {
	for _, v := range view {
		if v.ID == id {
			return v.Recommended
		}
	}
	return false
}

func Test_CatalogContainsExistingModelsAsDirect(t *testing.T) {
	ids := []EngineModel{
		EngineModelGeminiGemini2Dot5Flash, EngineModelGeminiGemini2Dot5Pro, EngineModelGeminiGemini2Dot0Flash, EngineModelGeminiGeminiProLatest,
		EngineModelOpenaiGPT5Dot2, EngineModelOpenaiGPT5Dot1, EngineModelOpenaiGPT5, EngineModelOpenaiGPT5Mini, EngineModelOpenaiGPT5Nano,
		EngineModelGrok3, EngineModelGrok3Mini,
	}
	for _, id := range ids {
		found := false
		for _, e := range catalog {
			if e.ID == id {
				found = true
				if e.Route != RouteDirect {
					t.Errorf("%s must be direct", id)
				}
			}
		}
		if !found {
			t.Errorf("%s missing from catalog", id)
		}
	}
}

func Test_CatalogViewPlatformManagedMatchesRoute(t *testing.T) {
	view := CatalogView()
	if len(view) != len(catalog) {
		t.Fatalf("view length mismatch. expect: %d, got: %d", len(catalog), len(view))
	}

	b, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(catalog) {
		t.Fatalf("decoded length mismatch. expect: %d, got: %d", len(catalog), len(decoded))
	}

	managed, direct := 0, 0
	for i, e := range catalog {
		expect := e.Route == RouteOpenRouter
		if expect {
			managed++
		} else {
			direct++
		}

		if view[i].ID != e.ID {
			t.Fatalf("order mismatch at %d. expect: %s, got: %s", i, e.ID, view[i].ID)
		}
		if view[i].PlatformManaged != expect {
			t.Errorf("%s: platform_managed mismatch. route: %s, expect: %v, got: %v", e.ID, e.Route, expect, view[i].PlatformManaged)
		}

		raw, ok := decoded[i]["platform_managed"]
		if !ok {
			t.Errorf("%s: platform_managed key missing in JSON", e.ID)
			continue
		}
		got, ok := raw.(bool)
		if !ok || got != expect {
			t.Errorf("%s: JSON platform_managed mismatch. route: %s, expect: %v, got: %v", e.ID, e.Route, expect, raw)
		}
	}
	if managed == 0 || direct == 0 {
		t.Errorf("catalog must contain both routes. managed: %d, direct: %d", managed, direct)
	}
}

func Test_CatalogCustomerFacingTextHasNoBannedTerms(t *testing.T) {
	banned := []string{"twilio", "vonage", "plivo", "messagebird", "fonoster", "alternative to", "openrouter", "zero data"}
	for _, e := range catalog {
		for field, text := range map[string]string{"label": e.Label, "description": e.Description} {
			lower := strings.ToLower(text)
			for _, b := range banned {
				if b == "openrouter" && e.Route == RouteCustomOpenRouter {
					continue
				}
				if strings.Contains(lower, b) {
					t.Errorf("%s: %s contains banned term %q: %q", e.ID, field, b, text)
				}
			}
		}
	}
}
