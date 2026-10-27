package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func Test_MergeEngineKey(t *testing.T) {
	tests := []struct {
		name       string
		storedMod  EngineModel
		storedKey  string
		newModel   EngineModel
		key        *string
		expectWrit *string
		expectKey  string
	}{
		{"omitted keeps the stored key", "openai.gpt-5", "dummy-stored", "openai.gpt-5", nil, nil, "dummy-stored"},
		{"omitted keeps the key inside the same service prefix", "openai.gpt-5", "dummy-stored", "openai.gpt-5-mini", nil, nil, "dummy-stored"},
		{"a blank value replaces like any other value", "openai.gpt-5", "dummy-stored", "openai.gpt-5", strPtr("  "), strPtr("  "), "  "},
		{"empty string deletes", "openai.gpt-5", "dummy-stored", "openai.gpt-5", strPtr(""), strPtr(""), ""},
		{"value replaces", "openai.gpt-5", "dummy-stored", "openai.gpt-5", strPtr("dummy-new"), strPtr("dummy-new"), "dummy-new"},
		{"omitted is dropped when the prefix changes to a keyed vendor", "openai.gpt-5", "dummy-stored", "gemini.gemini-2.5-pro", nil, strPtr(""), ""},
		{"omitted is dropped when the prefix changes to a custom openrouter model", "openai.gpt-5", "dummy-stored", "openrouter.vendor/model-a", nil, strPtr(""), ""},
		{"omitted is dropped when the prefix changes to a custom endpoint model", "openai.gpt-5", "dummy-stored", "customendpoint.m", nil, strPtr(""), ""},
		{"omitted is dropped for an unknown model with another prefix", "openai.gpt-5", "dummy-stored", "unknown.model", nil, strPtr(""), ""},
		{"omitted is kept when the prefix changes to a platform model", "openai.gpt-5", "dummy-stored", "anthropic.claude-haiku-4.5", nil, nil, "dummy-stored"},
		{"omitted is dropped when leaving a platform model for a keyed vendor", "anthropic.claude-haiku-4.5", "dummy-stored", "openai.gpt-5", nil, strPtr(""), ""},
		{"a model without a dot is its own prefix", "legacy", "dummy-stored", "legacy", nil, nil, "dummy-stored"},
		{"an empty new model counts as the stored one", "openai.gpt-5", "dummy-stored", "", nil, nil, "dummy-stored"},
		{"nothing stored stays empty", "openai.gpt-5", "", "openai.gpt-5", nil, nil, ""},
		{"prefix compare is case sensitive", "openai.gpt-5", "dummy-stored", "OpenAI.gpt-5", nil, strPtr(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := &AI{EngineModel: tt.storedMod, EngineKey: tt.storedKey}

			write, final := MergeEngineKey(stored, tt.newModel, tt.key)
			if final != tt.expectKey {
				t.Errorf("final key. expect: %q, got: %q", tt.expectKey, final)
			}
			if (write == nil) != (tt.expectWrit == nil) || (write != nil && *write != *tt.expectWrit) {
				t.Errorf("write. expect: %v, got: %v", tt.expectWrit, write)
			}
			if stored.EngineKey != tt.storedKey {
				t.Errorf("the stored value must not change, got %q", stored.EngineKey)
			}
		})
	}
}

// Test_servicePrefixKeyModeInvariant pins the property the omitted-key rule relies on: no service
// prefix mixes platform and non-platform catalog entries, so a key kept on a platform model can
// only move to a platform model of the same prefix.
func Test_servicePrefixKeyModeInvariant(t *testing.T) {
	platform := map[string]bool{}
	keyed := map[string]bool{}

	for _, m := range CatalogView() {
		prefix := servicePrefix(m.ID)
		if m.KeyMode == KeyModePlatform {
			platform[prefix] = true
		} else {
			keyed[prefix] = true
		}
	}

	if len(platform) == 0 || len(keyed) == 0 {
		t.Fatalf("the catalog must have both kinds. platform: %v, keyed: %v", platform, keyed)
	}
	for prefix := range platform {
		if keyed[prefix] {
			t.Errorf("service prefix %q has both platform and non-platform catalog entries", prefix)
		}
	}
}

func Test_IsPlatformModel(t *testing.T) {
	tests := []struct {
		model  EngineModel
		expect bool
	}{
		{"anthropic.claude-haiku-4.5", true},
		{"openai.gpt-5", false},
		{"openrouter.vendor/model-a", false},
		{"customendpoint.m", false},
		{"unknown.model", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsPlatformModel(tt.model); got != tt.expect {
			t.Errorf("IsPlatformModel(%q). expect: %v, got: %v", tt.model, tt.expect, got)
		}
	}
}

// Test_HasEngineKey_consistency: the same value comes out of the REST conversion, of the webhook
// payload built from the event copy, and of the timeline re-conversion of the stored event JSON.
func Test_HasEngineKey_consistency(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		expect bool
	}{
		{"key held", "dummy-key-value", true},
		{"no key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := &AI{EngineModel: "openai.gpt-5", EngineKey: tt.key}

			// REST
			rest := orig.ConvertWebhookMessage()
			if rest.HasEngineKey != tt.expect {
				t.Errorf("REST. expect: %v, got: %v", tt.expect, rest.HasEngineKey)
			}

			// event copy as publishAIEvent builds it
			c := *orig
			c.HasEngineKey = orig.EngineKey != "" || orig.HasEngineKey
			c.EngineKey = ""

			// webhook payload
			hook, err := c.CreateWebhookEvent()
			if err != nil {
				t.Fatalf("CreateWebhookEvent: %v", err)
			}
			var raw map[string]any
			if err := json.Unmarshal(hook, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got, ok := raw["has_engine_key"].(bool); !ok || got != tt.expect {
				t.Errorf("webhook. expect: %v, got: %v", tt.expect, raw["has_engine_key"])
			}
			if _, ok := raw["engine_key"]; ok || strings.Contains(string(hook), "dummy-key-value") {
				t.Errorf("the webhook payload must not carry the key: %s", hook)
			}

			// event bus JSON, read back as the timeline aggregation does
			bus, err := json.Marshal(&c)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(bus), "dummy-key-value") {
				t.Errorf("the event JSON must not carry the key: %s", bus)
			}
			var back AI
			if err := json.Unmarshal(bus, &back); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			re := back.ConvertWebhookMessage()
			if re.HasEngineKey != tt.expect {
				t.Errorf("timeline. expect has_engine_key: %v, got: %v", tt.expect, re.HasEngineKey)
			}
			reJSON, err := json.Marshal(re)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(reJSON), `"engine_key"`) || strings.Contains(string(reJSON), "dummy-key-value") {
				t.Errorf("the timeline conversion must not carry the key: %s", reJSON)
			}
		})
	}
}

func Test_WebhookMessage_hasEngineKeyAlwaysPresent(t *testing.T) {
	b, err := json.Marshal((&AI{}).ConvertWebhookMessage())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"has_engine_key":false`) {
		t.Errorf("has_engine_key must be present when false: %s", b)
	}
}
