package builderhandler

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// dailyWindow is the fixed window of the daily counter: 24 hours from the
// first turn, not a calendar day (design 4.3).
const dailyWindow = 24 * time.Hour

// Chat runs one Builder turn.
//
// ORDER (design 4.3): kill switch and key, ValidateRequest, semaphore,
// daily counter, model call. The order is what keeps a request that was never
// processed from spending the customer's daily allowance:
//
//   - a request that fails validation is never counted;
//   - a request refused because the process is busy is never counted;
//   - a request over the daily limit gives its semaphore slot back at once.
//
// Everything from the counter on IS counted, including a model failure: the
// counter is incremented before the call, so a customer retrying against a
// failing provider spends the allowance (plan deviation 5).
//
// The returned error never carries the customer's input, the engine's error
// text or the model's answer: a provider 4xx can echo the prompt, and the
// prompt is the customer's own business description. Only fixed reason codes
// and counts reach a log line.
func (h *builderHandler) Chat(ctx context.Context, customerID uuid.UUID, req *builder.ChatRequest) (resp *builder.ChatResponse, err error) {
	start := time.Now()
	log := logrus.WithFields(logrus.Fields{
		"func":        "Chat",
		"customer_id": customerID,
	})
	defer func() { promBuilderChatDuration.Observe(time.Since(start).Seconds()) }()

	if !h.opts.Enabled {
		return nil, h.fail(log, resultDisabled, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonDisabled, "the assistant builder is not enabled"))
	}
	if !h.opts.KeyConfigured {
		return nil, h.fail(log, resultUnavailable, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the assistant builder is not available"))
	}

	if errValidate := builder.ValidateRequest(req); errValidate != nil {
		// ValidateRequest builds its own *VoipbinError and never echoes input.
		return nil, h.fail(log, resultInvalidArgument, errValidate)
	}
	log = log.WithField("message_count", len(req.Messages))

	// Non-blocking acquire. The deferred release runs on every exit path below,
	// including a panic in the engine, so a few failures can never lock the
	// Builder out of this process.
	select {
	case h.sem <- struct{}{}:
	default:
		return nil, h.fail(log, resultBusy, cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "the assistant builder is busy, try again shortly"))
	}
	defer func() { <-h.sem }()

	count, errCount := h.cache.BuilderChatCountIncr(ctx, customerID, dailyWindow)
	if errCount != nil {
		// Fail CLOSED: an unmetered Builder is unbounded platform spend. The
		// Redis error text is not carried on, since it is not needed by the caller.
		log.WithField("error_kind", "counter").Errorf("Could not count the builder turn.")
		return nil, h.fail(log, resultUnavailable, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the assistant builder is not available"))
	}
	if count > int64(h.opts.DailyLimit) {
		return nil, h.fail(log, resultDailyLimit, cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "the daily limit of the assistant builder has been reached"))
	}

	res, errTurn := RunTurn(ctx, h.sender, h.cfg, req)
	// Tokens are recorded whether or not the answer was usable: the platform
	// paid for them.
	if res != nil {
		promBuilderTokensTotal.WithLabelValues("prompt").Add(float64(res.Usage.PromptTokens))
		promBuilderTokensTotal.WithLabelValues("completion").Add(float64(res.Usage.CompletionTokens))
	}
	if errTurn != nil {
		return nil, h.mapTurnError(log, errTurn)
	}

	promBuilderChatTotal.WithLabelValues(resultOK).Inc()
	log.Debug("Finished the builder turn.")

	return &builder.ChatResponse{
		Message:       res.Parsed.Message,
		Draft:         res.Parsed.Draft,
		Assumptions:   res.Parsed.Assumptions,
		DraftWarnings: res.Parsed.Warnings,
	}, nil
}

// mapTurnError turns RunTurn's sentinels into the reasons of design 4.7.
func (h *builderHandler) mapTurnError(log *logrus.Entry, err error) error {
	var llm *LLMError

	switch {
	case errors.Is(err, ErrTimeout):
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "the assistant builder took too long, try again"))

	case errors.As(err, &llm) && llm.Code == "timeout":
		// A deadline that belongs to the CALLER (for example the RPC timeout
		// api-manager waits with). RunTurn reports it as an LLM error; to the
		// client it is the same failure as the LLM deadline.
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "the assistant builder took too long, try again"))

	case errors.Is(err, ErrTruncated), errors.Is(err, ErrInvalidResponse):
		return h.fail(log, resultInvalidResponse, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "the assistant builder gave an unusable answer, try again"))

	case errors.Is(err, ErrLLM):
		// A provider error. Design 4.7 has no row for it; the plan (deviation 5)
		// maps it to BUILDER_RESPONSE_INVALID so the client shows "try again".
		// Only the fixed classification code is logged, never the error text.
		if llm != nil {
			log = log.WithField("llm_error", llm.Code)
		}
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "the assistant builder gave an unusable answer, try again"))

	default:
		// RunTurn only returns the sentinels above and plain programming errors
		// (nil sender or request). Do not carry the error text.
		return h.fail(log, resultLLMError, cerrors.Internal(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the assistant builder failed"))
	}
}

// fail counts the result, logs it without any input, and returns the error.
func (h *builderHandler) fail(log *logrus.Entry, result string, err error) error {
	promBuilderChatTotal.WithLabelValues(result).Inc()
	log.WithField("result", result).Info("The builder turn did not succeed.")
	return err
}
