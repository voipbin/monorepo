package listenhandler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-ai-manager/models/ai"
)

func Test_processV1AIModelsGet(t *testing.T) {
	h := &listenHandler{}

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/ai_models",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Wrong status code. expect: 200, got: %d", res.StatusCode)
	}
	if res.DataType != "application/json" {
		t.Errorf("Wrong data type. got: %s", res.DataType)
	}

	var items []map[string]any
	if err := json.Unmarshal(res.Data, &items); err != nil {
		t.Fatalf("Could not unmarshal the response. err: %v", err)
	}
	if len(items) != len(ai.CatalogView()) || len(items) == 0 {
		t.Fatalf("Wrong item count. expect: %d, got: %d", len(ai.CatalogView()), len(items))
	}
	for _, item := range items {
		for _, key := range []string{"id", "label", "vendor", "recommended", "tags", "description", "platform_managed", "key_mode"} {
			if _, ok := item[key]; !ok {
				t.Errorf("item %v misses key %s", item["id"], key)
			}
		}
	}

	// The custom entry is checked by key name: its id, label and prefix legitimately
	// contain "openrouter" (which itself contains "route").
	allowedOpenRouter := map[string]bool{"id": true, "label": true, "vendor": true, "description": true, "model_id_prefix": true}
	prefixed := 0
	for _, item := range items {
		if _, ok := item["model_id_prefix"]; ok {
			prefixed++
			for k, v := range item {
				lower := strings.ToLower(fmt.Sprint(v))
				if strings.Contains(lower, "slug") || strings.Contains(lower, "meta-llama/") {
					t.Errorf("custom entry field %s leaks internals", k)
				}
				if !allowedOpenRouter[k] && strings.Contains(lower, "openrouter") {
					t.Errorf("custom entry field %s must not mention openrouter", k)
				}
			}
			for _, k := range []string{"slug", "upstream_slug", "route", "custom_prefix"} {
				if _, ok := item[k]; ok {
					t.Errorf("custom entry leaks key %s", k)
				}
			}
			continue
		}

		raw, _ := json.Marshal(item)
		body := strings.ToLower(string(raw))
		for _, banned := range []string{"slug", "openrouter", "meta-llama/", "\"route\""} {
			if strings.Contains(body, banned) {
				t.Errorf("item %v leaks %q", item["id"], banned)
			}
		}
	}
	if prefixed != 1 {
		t.Errorf("exactly one entry must carry model_id_prefix. got: %d", prefixed)
	}
}

func Test_processV1AIModels_NonGetNotRouted(t *testing.T) {
	h := &listenHandler{}

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/ai_models",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("Wrong status code. expect: 404, got: %d", res.StatusCode)
	}
}

func Test_regV1AIModels_DoesNotCollideWithAIs(t *testing.T) {
	if regV1AIsGet.MatchString("/v1/ai_models") {
		t.Errorf("regV1AIsGet must not match /v1/ai_models")
	}
	if regV1AIs.MatchString("/v1/ai_models") {
		t.Errorf("regV1AIs must not match /v1/ai_models")
	}
	if regV1AIsID.MatchString("/v1/ai_models") {
		t.Errorf("regV1AIsID must not match /v1/ai_models")
	}
	if !regV1AIModels.MatchString("/v1/ai_models") {
		t.Errorf("regV1AIModels must match /v1/ai_models")
	}
	if regV1AIModels.MatchString("/v1/ais") {
		t.Errorf("regV1AIModels must not match /v1/ais")
	}
}
