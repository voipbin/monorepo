package builderhandler

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// Chat runs one Flow Builder turn.
//
// ORDER (design doc 5.1): key, ValidateRequest, semaphore, daily counter,
// model call, then the deterministic steps (assemble, validate). A request
// that was never processed never spends the customer's allowance. A model
// failure after the counter does spend it, as in the Assistant Builder.
//
// The returned error never carries the customer's input, the provider's error
// text or the model's answer; only fixed reason codes and counts are logged.
func (h *flowBuilderHandler) Chat(ctx context.Context, customerID uuid.UUID, req *flowbuilder.ChatRequest) (resp *flowbuilder.ChatResponse, err error) {
	start := time.Now()
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowChat",
		"customer_id": customerID,
	})
	defer func() { promFlowBuilderChatDuration.Observe(time.Since(start).Seconds()) }()

	if !h.opts.KeyConfigured {
		return nil, h.fail(log, resultUnavailable, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the flow builder is not available"))
	}

	if errValidate := flowbuilder.ValidateRequest(req); errValidate != nil {
		return nil, h.fail(log, resultInvalidArgument, errValidate)
	}
	log = log.WithField("message_count", len(req.Messages))

	// A request that can never be drafted is the caller's mistake and is
	// refused before it takes a semaphore slot or spends the allowance.
	allowed := FlowAllowedTypes(req.SupportedActionTypes)
	if len(allowed) == 0 {
		return nil, h.fail(log, resultInvalidArgument, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "supported_action_types has no usable action type"))
	}

	select {
	case h.sem <- struct{}{}:
	default:
		return nil, h.fail(log, resultBusy, cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "the flow builder is busy, try again shortly"))
	}
	defer func() { <-h.sem }()

	count, errCount := h.cache.BuilderFlowChatCountIncr(ctx, customerID, dailyWindow)
	if errCount != nil {
		// Fail CLOSED: an unmetered builder is unbounded platform spend.
		log.WithField("error_kind", "counter").Errorf("Could not count the flow builder turn.")
		return nil, h.fail(log, resultUnavailable, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the flow builder is not available"))
	}
	if count > int64(h.opts.DailyLimit) {
		return nil, h.fail(log, resultDailyLimit, cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "the daily limit of the flow builder has been reached"))
	}

	res, errTurn := RunFlowTurn(ctx, h.sender, h.cfg, req, allowed)
	if res != nil {
		promFlowBuilderTokensTotal.WithLabelValues("prompt").Add(float64(res.Usage.PromptTokens))
		promFlowBuilderTokensTotal.WithLabelValues("completion").Add(float64(res.Usage.CompletionTokens))
	}
	if errTurn != nil {
		return nil, h.mapTurnError(log, errTurn)
	}

	out := &flowbuilder.ChatResponse{
		Message:       res.Parsed.Message,
		Assumptions:   res.Parsed.Assumptions,
		DraftWarnings: res.Parsed.Warnings,
	}
	if res.Parsed.Graph != nil {
		draft, warnings := AssembleFlowDraft(*res.Parsed.Graph, allowedSet(allowed))
		out.Draft = draft
		out.DraftWarnings = append(out.DraftWarnings, warnings...)
		if draft == nil {
			out.Assumptions = nil // assumptions only travel with a draft
		} else {
			out.SensitiveNodes = sensitiveNodeIDs(draft)
		}
	}

	promFlowBuilderChatTotal.WithLabelValues(resultOK).Inc()
	log.Debug("Finished the flow builder turn.")
	return out, nil
}

// sensitiveNodeIDs lists the ids of nodes whose type the metadata marks
// sensitive. The decision is the registry's, never the model's.
func sensitiveNodeIDs(draft *flowbuilder.Draft) []uuid.UUID {
	var ids []uuid.UUID
	for _, a := range draft.Actions {
		t, _ := a["type"].(string)
		if meta, ok := fmaction.MetaByType[fmaction.Type(t)]; !ok || meta.Exposure != fmaction.ExposureSensitive {
			continue
		}
		id, _ := a["id"].(string)
		if u, err := uuid.FromString(id); err == nil {
			ids = append(ids, u)
		}
	}
	return ids
}

// mapTurnError turns RunFlowTurn's sentinels into the shared reasons.
func (h *flowBuilderHandler) mapTurnError(log *logrus.Entry, err error) error {
	var llm *LLMError

	switch {
	case errors.Is(err, ErrTimeout):
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "the flow builder took too long, try again"))

	case errors.As(err, &llm) && llm.Code == "timeout":
		// A deadline that belongs to the caller's context.
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "the flow builder took too long, try again"))

	case errors.Is(err, ErrTruncated), errors.Is(err, ErrInvalidResponse):
		return h.fail(log, resultInvalidResponse, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "the flow builder gave an unusable answer, try again"))

	case errors.Is(err, ErrLLM):
		if llm != nil {
			log = log.WithField("llm_error", llm.Code)
			if llm.Code == "auth" {
				log.Error("The flow builder's provider refused the platform key.")
			}
		}
		return h.fail(log, resultLLMError, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "the flow builder gave an unusable answer, try again"))

	default:
		return h.fail(log, resultLLMError, cerrors.Internal(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the flow builder failed"))
	}
}

// fail counts the result in the Flow series, logs it without any input, and
// returns the error.
func (h *flowBuilderHandler) fail(log *logrus.Entry, result string, err error) error {
	promFlowBuilderChatTotal.WithLabelValues(result).Inc()
	log.WithField("result", result).Info("The flow builder turn did not succeed.")
	return err
}
