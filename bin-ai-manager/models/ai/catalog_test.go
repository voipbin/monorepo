package ai

import (
	"encoding/json"
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
		if e.Route != RouteDirect && e.Route != RouteOpenRouter {
			t.Errorf("%s: unknown route %q", e.ID, e.Route)
		}
		for _, tag := range e.Tags {
			if tag != TagLowCost {
				t.Errorf("%s: unknown tag %q", e.ID, tag)
			}
		}
	}
}

func Test_CatalogPublicViewHidesInternals(t *testing.T) {
	b, err := json.Marshal(CatalogView())
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"slug", "route", "openrouter", "meta-llama/"} {
		if strings.Contains(strings.ToLower(string(b)), banned) {
			t.Errorf("public catalog leaks %q", banned)
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
				if strings.Contains(lower, b) {
					t.Errorf("%s: %s contains banned term %q: %q", e.ID, field, b, text)
				}
			}
		}
	}
}
