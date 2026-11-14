package configkeys

import "testing"

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
