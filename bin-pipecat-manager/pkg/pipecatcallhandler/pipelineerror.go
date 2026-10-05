package pipecatcallhandler

import (
	"regexp"
	"strings"

	"monorepo/bin-pipecat-manager/models/message"
)

// pipelineErrorLogMaxLen caps the raw error text written to the log. Provider error bodies can be
// long JSON documents; the first part carries the status/reason that matters.
const pipelineErrorLogMaxLen = 2000

// Classifier tiers for classifyPipelineError (VOIP-1542 design §3.1). All matching is done on the
// lower-cased error text and the first hit wins. Bare numbers and bare words such as
// "unauthorized" or "quota" are intentionally NOT matched on their own: request ids and URLs can
// contain them.
var (
	// tier 0: platform-side failures (never shown to the customer).
	pipelineErrorPrefixFunctionCall = "error executing function call ["
	pipelineErrorPrefixInternal     = "invalid rtvi transport message"

	// tier 1: structured provider status / reason tokens.
	pipelineErrorTokensAuthentication = []string{"api_key_invalid", "invalid_api_key", "permission_denied", "unauthenticated"}
	pipelineErrorTokensRateLimited    = []string{"resource_exhausted", "rate_limit_exceeded"}

	// tier 2: HTTP status phrases, anchored to the phrase rather than a bare number.
	pipelineErrorPhrasesAuthentication = []string{"401 unauthorized", "403 forbidden", "http 401", "http 403"}
	pipelineErrorRegexAuthentication   = regexp.MustCompile(`(status|code)["':= ]{0,4}(401|403)\b`)
	pipelineErrorPhrasesRateLimited    = []string{"429 too many requests"}
	pipelineErrorRegexRateLimited      = regexp.MustCompile(`(status|code)["':= ]{0,4}429\b`)

	// tier 3: provider free-text phrases, and the literal timeout strings pipecat emits for the
	// LLM services VoIPBin wires. Generic "timed out" / "deadline exceeded" text is NOT matched:
	// Google STT inactivity reconnects use it and must stay "unknown".
	pipelineErrorPhrasesAuthenticationText = []string{"api key not valid", "invalid api key", "incorrect api key", "invalid authentication"}
	pipelineErrorPhrasesRateLimitedText    = []string{"rate limit", "exceeded your current quota", "quota exceeded", "insufficient credits", "more credits"}
	pipelineErrorPhrasesTimeout            = []string{
		"llm completion timeout",                     // pipecat OpenAILLMService (openai, grok) httpx timeout
		"error during completion: request timed out", // openai APITimeoutError via OpenAILLMService
		"504 gateway timeout",                        // gemini HTTP error response (reason phrase form)
		"504 deadline_exceeded",                      // gemini in-stream error chunk
	}
)

// classifyPipelineError maps a raw RTVI error text to a category. Deterministic and pure.
func classifyPipelineError(raw string) message.ErrorCategory {
	s := strings.ToLower(raw)

	// tier 0
	if strings.HasPrefix(s, pipelineErrorPrefixFunctionCall) {
		return message.ErrorCategoryFunctionCall
	}
	if strings.HasPrefix(s, pipelineErrorPrefixInternal) {
		return message.ErrorCategoryInternal
	}

	// tier 1
	if containsAny(s, pipelineErrorTokensAuthentication) {
		return message.ErrorCategoryAuthentication
	}
	if containsAny(s, pipelineErrorTokensRateLimited) {
		return message.ErrorCategoryRateLimited
	}

	// tier 2
	if containsAny(s, pipelineErrorPhrasesAuthentication) || pipelineErrorRegexAuthentication.MatchString(s) {
		return message.ErrorCategoryAuthentication
	}
	if containsAny(s, pipelineErrorPhrasesRateLimited) || pipelineErrorRegexRateLimited.MatchString(s) {
		return message.ErrorCategoryRateLimited
	}

	// tier 3
	if containsAny(s, pipelineErrorPhrasesAuthenticationText) {
		return message.ErrorCategoryAuthentication
	}
	if containsAny(s, pipelineErrorPhrasesRateLimitedText) {
		return message.ErrorCategoryRateLimited
	}
	if containsAny(s, pipelineErrorPhrasesTimeout) {
		return message.ErrorCategoryTimeout
	}

	return message.ErrorCategoryUnknown
}

// shouldNotifyPipelineError decides whether a pipeline error becomes a customer-visible aicall
// notice (VOIP-1542 design §3.1). First match wins:
//  1. function_call / internal: never (platform-side bug, operator signal only).
//  2. fatal: always (the pipeline is being cancelled). Future-proofing: no service VoIPBin wires
//     today pushes fatal=true.
//  3. authentication / rate_limited / timeout: always (actionable, persistent or LLM-origin).
//  4. unknown: only for sessions without STT. In text sessions every error is LLM-origin; in voice
//     sessions a non-fatal unknown includes transient STT reconnects, which would be false alarms.
func shouldNotifyPipelineError(category message.ErrorCategory, fatal bool, hasSTT bool) bool {
	switch category {
	case message.ErrorCategoryFunctionCall, message.ErrorCategoryInternal:
		return false
	}

	if fatal {
		return true
	}

	switch category {
	case message.ErrorCategoryAuthentication, message.ErrorCategoryRateLimited, message.ErrorCategoryTimeout:
		return true
	case message.ErrorCategoryUnknown:
		return !hasSTT
	default:
		return false
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// truncateForLog shortens s to at most maxLen bytes for logging, on a rune boundary.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := maxLen
	for cut > 0 && !isRuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "...(truncated)"
}

func isRuneStart(b byte) bool {
	return b&0xC0 != 0x80
}
