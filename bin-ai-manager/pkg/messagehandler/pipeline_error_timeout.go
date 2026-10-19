package messagehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// eventPMPipecatcallTerminatedContactCase records a timeout notice when a contact_case turn was
// ended by the delayed terminate (defaultAITaskTimeout) while the user's message was still
// unanswered, so the end does not look like silence (VOIP-1633). The pipecat-manager stop path
// does not tell a delayed stop from any other stop, so the decision is made from the aicall state:
//
//  1. the aicall is not terminating or terminated (an explicit end is not a timeout),
//  2. the terminated pipecatcall is the aicall's current one (older turns and Insight listen turns
//     are excluded; this is re-read without the cache after the grace sleep because
//     SendReferenceTypeOthers interrupts the previous pipecatcall before it records the new id),
//  3. the newest non-notification message is a user message and the pipecatcall has no delivered
//     assistant reply (a turn that ran tools leaves a tool row and is not reported),
//  4. no pipeline_error notice exists for this pipecatcall and no timeout notice exists for the
//     aicall within pipelineErrorNoticeWindow.
//
// Lookup errors in 1 to 3 end the check without a notice (the behavior before this change). A scan
// error in 4 lets the notice through, like pipelineErrorNoticeRecent. It always returns nil so the
// event is not redelivered.
func (h *messageHandler) eventPMPipecatcallTerminatedContactCase(ctx context.Context, evt *pmpipecatcall.Pipecatcall, cached *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "eventPMPipecatcallTerminatedContactCase",
		"pipecatcall_id": evt.ID,
		"aicall_id":      cached.ID,
	})

	// cheap pre-filter on the cached row, before the grace sleep. Termination is one way, and a
	// stale pipecatcall id here at worst skips a notice.
	if cached.Status == aicall.StatusTerminating || cached.Status == aicall.StatusTerminated || cached.PipecatcallID != evt.ID {
		promBackstopReplyTotal.WithLabelValues("timeout_notice_skipped").Inc()
		return nil
	}

	// let an in-flight EventPMMessageBotLLM commit its reply row first.
	backstopGraceSleep(backstopGraceDelay)

	ac, err := h.reqHandler.AIV1AIcallGetSkipCache(ctx, cached.ID)
	if err != nil {
		log.Errorf("Could not re-read the aicall. err: %v", err)
		return nil
	}
	if ac.Status == aicall.StatusTerminating || ac.Status == aicall.StatusTerminated || ac.PipecatcallID != evt.ID {
		promBackstopReplyTotal.WithLabelValues("timeout_notice_skipped").Inc()
		return nil
	}

	unanswered, err := h.userMessageUnanswered(ctx, ac.ID, evt.ID)
	if err != nil {
		log.Errorf("Could not check the unanswered user message. err: %v", err)
		return nil
	}
	if !unanswered {
		promBackstopReplyTotal.WithLabelValues("timeout_notice_skipped").Inc()
		return nil
	}

	if h.pipelineErrorNoticeBlocks(ctx, ac.ID, evt.ID, pmmessage.ErrorCategoryTimeout) {
		promBackstopReplyTotal.WithLabelValues("timeout_notice_skipped").Inc()
		return nil
	}

	activeAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
	if errCreate := h.createPipelineErrorNotice(ctx, ac.CustomerID, ac.ID, ac.ActiveflowID, evt.ID, pmmessage.ErrorCategoryTimeout, true, activeAIID); errCreate != nil {
		promBackstopReplyTotal.WithLabelValues("timeout_notice_failed").Inc()
		return nil
	}
	promBackstopReplyTotal.WithLabelValues("timeout_notice_created").Inc()
	log.Infof("Recorded a timeout notice for a turn that ended without an answer.")
	return nil
}

// userMessageUnanswered reports whether the newest non-notification message of the aicall is a user
// message and the pipecatcall has no delivered assistant reply.
func (h *messageHandler) userMessageUnanswered(ctx context.Context, aicallID, pipecatcallID uuid.UUID) (bool, error) {
	msgs, err := h.db.MessageList(ctx, 1, "", map[message.Field]any{
		message.FieldAIcallID: aicallID,
		message.FieldDeleted:  false,
		message.FieldRole:     commondatabasehandler.NotEq{Value: string(message.RoleNotification)},
	})
	if err != nil {
		return false, err
	}
	if len(msgs) == 0 || msgs[0].Role != message.RoleUser {
		return false, nil
	}

	replied, err := h.db.MessageAssistantReplyExists(ctx, pipecatcallID)
	if err != nil {
		return false, err
	}
	return !replied, nil
}

// pipelineErrorNoticeBlocks reports whether a new notice of the given category must not be created
// for the pipecatcall: a pipeline_error notice (any category) already exists for the pipecatcall, or
// one of the same category exists on the aicall within pipelineErrorNoticeWindow. Only the newest
// pipelineErrorNoticeScanSize notification rows are scanned. Errors count as no match (fail-open).
func (h *messageHandler) pipelineErrorNoticeBlocks(ctx context.Context, aicallID, pipecatcallID uuid.UUID, category pmmessage.ErrorCategory) bool {
	msgs, err := h.db.MessageList(ctx, pipelineErrorNoticeScanSize, "", map[message.Field]any{
		message.FieldAIcallID: aicallID,
		message.FieldRole:     message.RoleNotification,
		message.FieldDeleted:  false,
	})
	if err != nil {
		logrus.WithField("aicall_id", aicallID).Warnf("Could not list notification messages for the dedup check. err: %v", err)
		return false
	}

	cutoff := h.utilHandler.TimeNow().Add(-pipelineErrorNoticeWindow)
	for _, m := range msgs {
		var n PipelineErrorNotice
		if errUnmarshal := json.Unmarshal([]byte(m.Content), &n); errUnmarshal != nil {
			continue
		}
		if n.Type != NotificationTypePipelineError {
			continue
		}
		if n.PipecatcallID == pipecatcallID {
			return true
		}
		if n.Category == category && m.TMCreate != nil && !m.TMCreate.Before(cutoff) {
			return true
		}
	}
	return false
}
