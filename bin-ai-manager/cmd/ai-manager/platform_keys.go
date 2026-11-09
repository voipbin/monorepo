package main

import (
	"monorepo/bin-common-handler/pkg/confighandler"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
)

// platformKeys are the suppliers of the platform LLM keys, one per consumer. Each one reads the key
// when it is called (nothing is kept), so a changed key is used by the next call.
type platformKeys struct {
	// conversation is the key of the conversational engine (the OpenAI default endpoint).
	conversation aiconfig.KeyFunc
	// summary is the key of the summary engine: the one of the provider at its base URL.
	summary aiconfig.KeyFunc
	// analysis is the key of the analysis engine, which the Builder shares: the one of the
	// provider at its base URL.
	analysis aiconfig.KeyFunc
	// gemini is the Gemini key of the handlers that always call the Gemini endpoint (audit and
	// prompt proposal).
	gemini aiconfig.KeyFunc
}

// newPlatformKeys picks the supplier of each consumer. It is the one place where a consumer is
// tied to a key, apart from the provider rule itself (aiconfig.KeyForBaseURL).
func newPlatformKeys(h confighandler.Handler, cfg *config.Config) platformKeys {
	return platformKeys{
		conversation: aiconfig.KeyForBaseURL(h, ""),
		summary:      aiconfig.KeyForBaseURL(h, cfg.SummaryEngineBaseURL),
		analysis:     aiconfig.KeyForBaseURL(h, cfg.AnalysisEngineBaseURL),
		gemini:       aiconfig.GeminiKey(h),
	}
}
