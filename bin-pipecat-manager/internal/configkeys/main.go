// Package configkeys declares the configuration keys of bin-pipecat-manager.
//
// The keys are the platform keys of the vendors that the python runner talks to. The Go service
// resolves the keys that a session uses when the session starts and hands them to the runner in
// the request (platform_keys). The runner never reads the database. A key without a stored row
// falls back to the environment variable of the same name, and the runner falls back to its own
// environment when the request has no value.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "pipecat_configs"
	HistoryTable = "pipecat_config_histories"
)

// Names of the keys. The name is also the key of the platform_keys map in the request to the
// runner, so it is part of the wire contract and never changes. A name never changes its kind: a
// stored row and its history would keep the old meaning.
const (
	KeyOpenAIAPIKey      = "openai.api_key"
	KeyDeepgramAPIKey    = "deepgram.api_key"
	KeyCartesiaAPIKey    = "cartesia.api_key"
	KeyXAIAPIKey         = "xai.api_key"
	KeyElevenLabsAPIKey  = "elevenlabs.api_key"
	KeyGoogleAPIKey      = "google.api_key"      // Gemini LLM only. Google STT and TTS use a service account.
	KeyOpenRouterAPIKey  = "openrouter.api_key"  // the platform OpenRouter key only, never a customer key.
	KeyLocalLLMEndpoints = "local_llm.endpoints" // JSON object text, see the runner.
)

const (
	// maxKeyLen is the longest accepted vendor key in bytes.
	maxKeyLen = 512

	// maxEndpointsLen is the longest accepted local endpoints text in bytes. It is the longest
	// plaintext that the secret column holds (2048 bytes minus nonce, tag and framing).
	maxEndpointsLen = 2020
)

// Keys returns the declarations. Every key is a secret. Env is the name of the variable that the
// runner reads today, so a key without a stored row is read from the environment of this
// process (and the runner falls back to its own environment when the request has no value).
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyOpenAIAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "OPENAI_API_KEY", Description: "OpenAI API key of the platform. Used by sessions that run an OpenAI model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyDeepgramAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "DEEPGRAM_API_KEY", Description: "Deepgram API key of the platform. Used by sessions that transcribe with Deepgram.", MaxLen: maxKeyLen},
		{Name: KeyCartesiaAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "CARTESIA_API_KEY", Description: "Cartesia API key of the platform. Used by sessions that speak with Cartesia.", MaxLen: maxKeyLen},
		{Name: KeyXAIAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "XAI_API_KEY", Description: "xAI API key of the platform. Used by sessions that run a Grok model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyElevenLabsAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "ELEVENLABS_API_KEY", Description: "ElevenLabs API key of the platform. Used by sessions that speak with ElevenLabs.", MaxLen: maxKeyLen},
		{Name: KeyGoogleAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "GOOGLE_API_KEY", Description: "Google API key of the platform for Gemini models. Used by sessions that run a Gemini model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyOpenRouterAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "OPENROUTER_API_KEY", Description: "OpenRouter API key of the platform. Used by platform OpenRouter models only; a key of the customer is never used for them.", MaxLen: maxKeyLen},
		{Name: KeyLocalLLMEndpoints, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "LOCAL_LLM_ENDPOINTS", Description: "One line JSON object that maps a local endpoint name to its base_url and api_key. Used by sessions that run a local model.", MaxLen: maxEndpointsLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
