// Package configkeys names the platform keys of the vendors that the python runner talks to.
//
// The runner reads the keys from the environment variables of its own container (the names are in
// scripts/pipecat/run.py); the Go service reads no key and sends none in the request. The names
// below are the keys of the platform_keys map of the runner request (the mapping of an llm, stt or
// tts type to its name is in pkg/pipecatcallhandler/platformkeys.go), so they are part of the wire
// contract and never change.
package configkeys

// Names of the keys.
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
