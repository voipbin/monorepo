package main

import (
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// builderSettings turns the service configuration into the Builder's two
// settings structs.
//
// The Builder reuses the analysis engine (same base URL, same key), so the key
// is "configured" only when that engine has one: the Google key when the base
// URL is Gemini, the OpenAI key otherwise. This is the same selection the
// analysis engine makes in run(); keeping it in one place means the two cannot
// disagree about whether a key exists.
//
// HARD CONSTRAINT H1: the platform key is read here and handed to the engine,
// nothing else. It is never written to a customer's AI row.
func builderSettings(cfg *config.Config) (builderhandler.Config, builderhandler.Options) {
	keyConfigured := cfg.EngineKeyChatGPT != ""
	if strings.Contains(cfg.AnalysisEngineBaseURL, "generativelanguage") {
		keyConfigured = cfg.GoogleAPIKey != ""
	}

	bc := builderhandler.DefaultConfig()
	bc.Model = cfg.AIBuilderModel
	bc.ReasoningEffort = cfg.AIBuilderReasoningEffort
	bc.MaxOutputTokens = cfg.AIBuilderMaxOutputTokens
	bc.LLMTimeout = time.Duration(cfg.AIBuilderLLMTimeoutSeconds) * time.Second

	return bc, builderhandler.Options{
		Enabled:       cfg.AIBuilderEnabled,
		KeyConfigured: keyConfigured,
		DailyLimit:    cfg.AIBuilderDailyLimit,
		MaxConcurrent: cfg.AIBuilderMaxConcurrent,
	}
}
