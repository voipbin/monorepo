package main

import (
	"context"
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// builderStartupKeyTimeout bounds the one key read of the startup notes.
const builderStartupKeyTimeout = 5 * time.Second

// builderSettings turns the service configuration into the Builder's two
// settings structs.
//
// The Builder reuses the analysis engine (same base URL, same key), so the key
// is "configured" only when that engine has one. analysisKey is the supplier of that
// key, the same function the engine calls (aiconfig.KeyForBaseURL), so the availability check
// and the model call can not disagree about which key the Builder needs. The key is read
// whenever the check runs, not once at startup.
//
// HARD CONSTRAINT H1: the platform key is read here and handed to the engine,
// nothing else. It is never written to a customer's AI row.
func builderSettings(cfg *config.Config, analysisKey aiconfig.KeyFunc) (builderhandler.Config, builderhandler.Options) {
	bc := builderhandler.DefaultConfig()
	bc.Model = cfg.AIBuilderModel
	bc.ReasoningEffort = cfg.AIBuilderReasoningEffort
	bc.MaxOutputTokens = cfg.AIBuilderMaxOutputTokens
	bc.LLMTimeout = time.Duration(cfg.AIBuilderLLMTimeoutSeconds) * time.Second

	return bc, builderhandler.Options{
		KeyConfigured: func(ctx context.Context) (bool, error) {
			return aiconfig.Configured(ctx, analysisKey)
		},
		DailyLimit:    cfg.AIBuilderDailyLimit,
		MaxConcurrent: cfg.AIBuilderMaxConcurrent,
	}
}

// builderStartupNotes lists what an operator should know at startup about the
// Builder (design 4.6). infos are notices (a missing key: the Builder is
// optional and many deploys never use it) and warns are things that look wrong.
//
// keyConfigured and errKey are the result of the key check at startup. The key can be set or
// changed later with ai-control config set, so this is the state at startup only.
//
// These cases are checked. A missing key otherwise fails silently: the status
// route just reports available=false and the cause is nowhere in the logs. A key that can not be
// read is a warning. When the key is missing nothing else is reported, since no call can run. A
// model name that does not match the provider does not affect the status route and
// only fails when a turn is run, with a provider error.
//   - no key for the provider the analysis engine points at;
//   - a model name that does not look like it belongs to that provider (a Gemini
//     model name on an OpenAI base URL after an OpenAI rollback, or the other way
//     round). This is a hint, not a rule: it only compares the name's prefix.
func builderStartupNotes(cfg *config.Config, keyConfigured bool, errKey error) (infos, warns []string) {
	if errKey != nil {
		return nil, []string{"the assistant builder key could not be read at startup (" + errKey.Error() + "); the builder reports itself unavailable until it can be read"}
	}

	if !keyConfigured {
		return []string{"the optional assistant builder is unavailable because the analysis engine has no API key (the Gemini key for a Gemini base URL, the OpenAI compatible key otherwise; set it in the environment or with ai-control config set); the admin UI hides it until one is set"}, nil
	}

	isGeminiURL := aiconfig.IsGeminiBaseURL(cfg.AnalysisEngineBaseURL)
	isGeminiModel := strings.HasPrefix(strings.ToLower(cfg.AIBuilderModel), "gemini")
	if isGeminiURL != isGeminiModel {
		warns = append(warns, "ai_builder_model does not look like it belongs to the provider of analysis_engine_base_url; check both")
	}

	return nil, warns
}
