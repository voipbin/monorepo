// Package configkeys declares the configuration keys of bin-pipecat-manager.
//
// The keys are the platform keys of the vendors that the python runner talks to. The runner reads
// them from the environment variables of its own container (the names are in scripts/pipecat/run.py),
// not from these stored values, and the Go service sends no key in the request. The declarations
// stay for the stored settings code (the config commands and the admin API) until it is removed.
// The keys declare no environment variable and have no default.
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

// Keys returns the declarations. Every key is a secret. A key declares no environment variable:
// the stored settings handler resolves stored rows only.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyOpenAIAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "OpenAI API key of the platform. Used by sessions that run an OpenAI model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyDeepgramAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Deepgram API key of the platform. Used by sessions that transcribe with Deepgram.", MaxLen: maxKeyLen},
		{Name: KeyCartesiaAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Cartesia API key of the platform. Used by sessions that speak with Cartesia.", MaxLen: maxKeyLen},
		{Name: KeyXAIAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "xAI API key of the platform. Used by sessions that run a Grok model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyElevenLabsAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "ElevenLabs API key of the platform. Used by sessions that speak with ElevenLabs.", MaxLen: maxKeyLen},
		{Name: KeyGoogleAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Google API key of the platform for Gemini models. Used by sessions that run a Gemini model without a key of the customer.", MaxLen: maxKeyLen},
		{Name: KeyOpenRouterAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "OpenRouter API key of the platform. Used by platform OpenRouter models only; a key of the customer is never used for them.", MaxLen: maxKeyLen},
		{Name: KeyLocalLLMEndpoints, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "One line JSON object that maps a local endpoint name to its base_url and api_key. Used by sessions that run a local model.", MaxLen: maxEndpointsLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
