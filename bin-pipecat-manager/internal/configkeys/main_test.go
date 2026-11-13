package configkeys

import (
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/confighandler"
)

func Test_NewRegistry(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(reg.Keys()) != 8 {
		t.Errorf("Wrong match. expect: 8 keys, got: %d", len(reg.Keys()))
	}
}

// Test_goldenKinds pins the name, kind, type, length and environment name of every key. A key
// name is never moved to another kind: rows and history that exist under the old meaning would be
// read as the new one. The environment name is what makes "no row, value in the environment"
// work, so a typo here is a regression, and it has to be the name that the runner reads.
func Test_goldenKinds(t *testing.T) {
	expect := []struct {
		name   string
		env    string
		maxLen int
	}{
		{"openai.api_key", "OPENAI_API_KEY", 512},
		{"deepgram.api_key", "DEEPGRAM_API_KEY", 512},
		{"cartesia.api_key", "CARTESIA_API_KEY", 512},
		{"xai.api_key", "XAI_API_KEY", 512},
		{"elevenlabs.api_key", "ELEVENLABS_API_KEY", 512},
		{"google.api_key", "GOOGLE_API_KEY", 512},
		{"openrouter.api_key", "OPENROUTER_API_KEY", 512},
		{"local_llm.endpoints", "LOCAL_LLM_ENDPOINTS", 2020},
	}

	keys := Keys()
	if len(keys) != len(expect) {
		t.Fatalf("Wrong match. expect: %d keys, got: %d", len(expect), len(keys))
	}
	for i, e := range expect {
		k := keys[i]
		if k.Name != e.name || k.Env != e.env || k.MaxLen != e.maxLen {
			t.Errorf("Wrong match. key %d expect: %+v, got: %+v", i, e, k)
		}
		if k.Kind != confighandler.KindSecret || k.Type != confighandler.TypeString {
			t.Errorf("Wrong match. %s must be a secret string, got: %v %v", k.Name, k.Kind, k.Type)
		}
		if k.Default != nil {
			t.Errorf("Wrong match. %s must have no default", k.Name)
		}
		if k.Group != "" {
			t.Errorf("Wrong match. %s must have no group, got: %q", k.Name, k.Group)
		}
	}
}

// The key names are the keys of platform_keys in the request to the runner.
func Test_wireNames(t *testing.T) {
	names := map[string]string{
		KeyOpenAIAPIKey:      "openai.api_key",
		KeyDeepgramAPIKey:    "deepgram.api_key",
		KeyCartesiaAPIKey:    "cartesia.api_key",
		KeyXAIAPIKey:         "xai.api_key",
		KeyElevenLabsAPIKey:  "elevenlabs.api_key",
		KeyGoogleAPIKey:      "google.api_key",
		KeyOpenRouterAPIKey:  "openrouter.api_key",
		KeyLocalLLMEndpoints: "local_llm.endpoints",
	}
	for got, expect := range names {
		if got != expect {
			t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
		}
	}
}

// Every key has a one line description for operators, and it never names a value.
func Test_descriptions(t *testing.T) {
	for _, k := range Keys() {
		if k.Description == "" {
			t.Errorf("Wrong match. %s has no description", k.Name)
		}
		if strings.ContainsAny(k.Description, "\r\n") || len(k.Description) > 300 {
			t.Errorf("Wrong match. %s: the description must be one short line, got: %q", k.Name, k.Description)
		}
	}
}

func Test_tableNames(t *testing.T) {
	if Table != "pipecat_configs" || HistoryTable != "pipecat_config_histories" {
		t.Errorf("Wrong match. the table names are part of the stored secret contract: %s %s", Table, HistoryTable)
	}
}
