// Package configkeys declares the configuration keys of bin-ai-manager.
//
// The two keys are the platform LLM API keys. They are stored in the database only. They are
// independent keys on purpose: which of them a call uses is decided by the provider of the call
// (see aiconfig.KeyForBaseURL), and they are rotated separately.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "ai_configs"
	HistoryTable = "ai_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	// KeyOpenAICompatAPIKey is the key of an OpenAI compatible provider (OpenAI, Grok and so on).
	// The name is provider neutral because one key serves all of them.
	KeyOpenAICompatAPIKey = "llm.openai_compat.api_key"

	// KeyGeminiAPIKey is the Google API key of the Gemini endpoint.
	KeyGeminiAPIKey = "llm.gemini.api_key"
)

// maxSecretLen is the longest accepted API key in bytes.
const maxSecretLen = 512

// Keys returns the declarations. The values are stored in the database only: a key has no
// environment name, so a key without a row is not configured.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyOpenAICompatAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the OpenAI compatible provider that serves the conversation engine, and the summary, analysis and builder LLM calls when their base URL is not the Gemini endpoint.", MaxLen: maxSecretLen},
		{Name: KeyGeminiAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Google API key of the Gemini endpoint (audit and prompt proposal, and summary, analysis and builder when their base URL is the Gemini endpoint).", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
