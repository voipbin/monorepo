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

// builderStartupWarnings lists what an operator should know at startup when the
// Builder is switched on but cannot work as intended (design 4.6). It returns
// nothing while the Builder is off, so an untouched deploy logs nothing new.
//
// Two cases are checked, both because they otherwise fail silently: the status
// route just reports available=false and the cause is nowhere in the logs.
//   - no key for the provider the analysis engine points at;
//   - a model name that does not look like it belongs to that provider (a Gemini
//     model name on an OpenAI base URL after an OpenAI rollback, or the other way
//     round). This is a hint, not a rule: it only compares the name's prefix.
func builderStartupWarnings(cfg *config.Config, opts builderhandler.Options) []string {
	if !opts.Enabled {
		return nil
	}

	var out []string
	if !opts.KeyConfigured {
		out = append(out, "AI_BUILDER_ENABLED is true but the analysis engine has no API key (GOOGLE_API_KEY for a Gemini base URL, ENGINE_KEY_CHATGPT otherwise); the builder will report itself unavailable")
	}

	isGeminiURL := strings.Contains(cfg.AnalysisEngineBaseURL, "generativelanguage")
	isGeminiModel := strings.HasPrefix(strings.ToLower(cfg.AIBuilderModel), "gemini")
	if isGeminiURL != isGeminiModel {
		out = append(out, "ai_builder_model does not look like it belongs to the provider of analysis_engine_base_url; check both")
	}

	return out
}
