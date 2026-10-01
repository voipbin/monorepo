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
// URL is Gemini, the OpenAI key otherwise. This is the same rule as the
// analysisKey selection in run(), written a second time here, so a change to
// one must be made in the other.
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
		KeyConfigured: keyConfigured,
		DailyLimit:    cfg.AIBuilderDailyLimit,
		MaxConcurrent: cfg.AIBuilderMaxConcurrent,
	}
}

// builderStartupWarnings lists what an operator should know at startup about the
// Builder (design 4.6). The first entry is informational and the rest are
// warnings; the caller logs the missing key at info level because the Builder is
// optional and many deploys never use it.
//
// Two cases are checked. A missing key otherwise fails silently: the status
// route just reports available=false and the cause is nowhere in the logs. When
// the key is missing nothing else is reported, since no call can run. A model
// name that does not match the provider does not affect the status route and
// only fails when a turn is run, with a provider error.
//   - no key for the provider the analysis engine points at;
//   - a model name that does not look like it belongs to that provider (a Gemini
//     model name on an OpenAI base URL after an OpenAI rollback, or the other way
//     round). This is a hint, not a rule: it only compares the name's prefix.
func builderStartupWarnings(cfg *config.Config, opts builderhandler.Options) []string {
	if !opts.KeyConfigured {
		return []string{"the optional assistant builder is unavailable because the analysis engine has no API key (GOOGLE_API_KEY for a Gemini base URL, ENGINE_KEY_CHATGPT otherwise); the admin UI hides it until one is set"}
	}

	var out []string

	isGeminiURL := strings.Contains(cfg.AnalysisEngineBaseURL, "generativelanguage")
	isGeminiModel := strings.HasPrefix(strings.ToLower(cfg.AIBuilderModel), "gemini")
	if isGeminiURL != isGeminiModel {
		out = append(out, "ai_builder_model does not look like it belongs to the provider of analysis_engine_base_url; check both")
	}

	return out
}
