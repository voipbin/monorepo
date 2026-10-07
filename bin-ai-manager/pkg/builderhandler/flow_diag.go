package builderhandler

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/flowbuilder"
)

// Diagnostic log of one Flow Builder model call (VOIP-1576).
//
// The line carries counts, sizes, durations, booleans, configured constants
// and fixed classes only. It never carries the customer's messages, the
// model's answer, the draft content, the system prompt, the schema text or
// any provider error text.

// The outcome values of the diagnostic line. Provider errors are
// "llm_" plus a ClassifyLLMError code (llm_timeout is the caller's context
// deadline, a different thing from outcomeTimeout, the handler's own one).
const (
	outcomeOK              = "ok"
	outcomeTimeout         = "timeout"
	outcomeTruncated       = "truncated"
	outcomeInvalidResponse = "invalid_response"
	outcomeError           = "error"
)

// flowDraftFacts are the facts about the draft that are known only after the
// answer has been assembled. They are only meaningful for the ok outcome.
type flowDraftFacts struct {
	hasDraft       bool
	draftDiscarded bool
	emptyDraft     bool
}

// flowOutcome classifies the error of RunFlowTurn in the same order as
// mapTurnError, so the two never disagree about what happened.
func flowOutcome(err error) string {
	var llm *LLMError

	switch {
	case err == nil:
		return outcomeOK
	case errors.Is(err, ErrTimeout):
		return outcomeTimeout
	case errors.As(err, &llm) && llm.Code == "timeout":
		return "llm_timeout"
	case errors.Is(err, ErrTruncated):
		return outcomeTruncated
	case errors.Is(err, ErrInvalidResponse):
		return outcomeInvalidResponse
	case errors.Is(err, ErrLLM) && llm != nil:
		return "llm_" + llm.Code
	default:
		return outcomeError
	}
}

// normalizeFinishReason keeps the finish reason to a closed set, because the
// provider's string is not under our control.
func normalizeFinishReason(reason string) string {
	switch reason {
	case "":
		return "none"
	case "stop", "length", "content_filter", "tool_calls", "function_call":
		return reason
	default:
		return "other"
	}
}

// hasWholeWarning reports whether list holds s as a whole entry. Matching is by
// whole-string equality, never by substring, because other warnings carry
// labels supplied by the model.
func hasWholeWarning(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// logModelCall writes the one diagnostic line of a model call. A nil result
// means the model was never called (the guard returns of RunFlowTurn), so
// nothing is written. It never changes what Chat returns.
func (h *flowBuilderHandler) logModelCall(log *logrus.Entry, res *FlowTurnResult, errTurn error, start, callStart time.Time, facts *flowDraftFacts) {
	if res == nil {
		return
	}

	fields := logrus.Fields{
		"outcome":               flowOutcome(errTurn),
		"elapsed_ms":            res.Diag.Elapsed.Milliseconds(),
		"build_ms":              res.Diag.BuildElapsed.Milliseconds(),
		"pre_call_ms":           callStart.Sub(start).Milliseconds(),
		"chat_ms":               time.Since(start).Milliseconds(),
		"finish_reason":         normalizeFinishReason(res.FinishReason),
		"prompt_tokens":         res.Usage.PromptTokens,
		"completion_tokens":     res.Usage.CompletionTokens,
		"response_chars":        res.Diag.ResponseChars,
		"system_chars":          res.Diag.SystemChars,
		"request_chars":         res.Diag.RequestChars,
		"schema_bytes":          res.Diag.SchemaBytes,
		"allowed_types":         res.Diag.AllowedTypes,
		"user_turns":            res.Diag.UserTurns,
		"history_messages":      res.Diag.HistoryMessages,
		"current_draft_present": res.Diag.CurrentDraftPresent,
		"model":                 h.cfg.Model,
		"reasoning_effort":      h.cfg.ReasoningEffort,
		"max_tokens":            h.cfg.MaxOutputTokens,
		"llm_timeout_ms":        h.cfg.LLMTimeout.Milliseconds(),
		"json_mode":             flowJSONModeName(h.cfg.JSONMode),
	}
	if res.Diag.InvalidKind != "" {
		fields["invalid_kind"] = res.Diag.InvalidKind
	}
	if facts != nil {
		fields["has_draft"] = facts.hasDraft
		fields["draft_discarded"] = facts.draftDiscarded
		fields["empty_draft"] = facts.emptyDraft
	}

	log.WithFields(fields).Info("The flow builder model call finished.")
}

// flowDraftFactsOf derives the draft facts of a successful turn.
func flowDraftFactsOf(parsed *FlowParsed, out *flowbuilder.ChatResponse) *flowDraftFacts {
	return &flowDraftFacts{
		hasDraft:       out.Draft != nil,
		draftDiscarded: parsed != nil && hasWholeWarning(parsed.Warnings, WarnDraftDiscarded),
		emptyDraft:     hasWholeWarning(out.DraftWarnings, flowbuilder.WarningEmptyDraft),
	}
}
