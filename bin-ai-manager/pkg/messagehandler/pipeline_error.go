package messagehandler

import (
	"context"
	"encoding/json"
	"time"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// NotificationTypePipelineError is the `type` of a role=notification message recording a pipeline
// error on an aicall (VOIP-1542).
const NotificationTypePipelineError = "pipeline_error"

const (
	// pipelineErrorNoticeWindow bounds repeated notices from Insight listen turns (foreign
	// pipecatcalls): at most one notice per category per aicall within this window.
	pipelineErrorNoticeWindow = 10 * time.Minute

	// pipelineErrorNoticeScanSize is how many recent notification rows are scanned for the
	// window check.
	pipelineErrorNoticeScanSize = 50
)

// pipelineErrorNoticeText is the fixed, plain-text, customer-facing sentence per category. The
// RTVI error does not say which service (LLM, STT or TTS) failed, nor whether the key was the
// customer's or the platform's, so the wording names the failure class without blaming a key.
var pipelineErrorNoticeText = map[pmmessage.ErrorCategory]string{
	pmmessage.ErrorCategoryAuthentication: "An AI service provider rejected the credentials or denied access. If this AI uses a custom engine key, verify that the key is valid and permitted for the selected model.",
	pmmessage.ErrorCategoryRateLimited:    "An AI service provider rejected the request due to a rate limit or quota. Try again later.",
	pmmessage.ErrorCategoryTimeout:        "The AI language model did not respond in time.",
	pmmessage.ErrorCategoryUnknown:        "An AI service provider returned an error.",
}

// PipelineErrorNotice is the JSON content of a pipeline_error notification message.
type PipelineErrorNotice struct {
	Type          string                  `json:"type"`
	Category      pmmessage.ErrorCategory `json:"category"`
	Fatal         bool                    `json:"fatal"`
	PipecatcallID uuid.UUID               `json:"pipecatcall_id"`
	Message       string                  `json:"message"`
}

// EventPMPipelineError records a pipeline error reported by pipecat-manager as a role=notification
// message on the owning aicall, so it reaches the admin timeline and the aimessage_created webhook
// instead of looking like silence (VOIP-1542). Notification rows are excluded from the conversing
// AI's prompt context by getPipecatcallMessages.
//
// Every failure path is fail-open toward creating the row: a missed error notice is worse than a
// duplicate one.
func (h *messageHandler) EventPMPipelineError(ctx context.Context, evt *pmmessage.PipelineErrorEvent) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "EventPMPipelineError",
		"pipecatcall_id": evt.PipecatcallID,
		"aicall_id":      evt.PipecatcallReferenceID,
		"category":       evt.Category,
	})

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		// only aicalls have a message timeline.
		return
	}

	text, ok := pipelineErrorNoticeText[evt.Category]
	if !ok {
		// function_call / internal are never published by pipecat-manager; anything else unknown
		// to this version is shown with the generic sentence.
		text = pipelineErrorNoticeText[pmmessage.ErrorCategoryUnknown]
	}

	activeAIID := uuid.Nil
	if h.reqHandler != nil {
		ac, errGet := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
		if errGet != nil {
			log.Warnf("Could not get the aicall. Creating the notice without dedup. err: %v", errGet)
		} else {
			if h.isForeignPipecatcall(ac, evt.PipecatcallID) {
				ac = h.confirmForeignPipecatcall(ctx, ac, evt.PipecatcallID)
				if h.isForeignPipecatcall(ac, evt.PipecatcallID) && h.pipelineErrorNoticeRecent(ctx, ac.ID, evt.Category) {
					log.Debugf("Skipping a repeated pipeline error notice from a foreign pipecatcall within the window.")
					return
				}
			}
			activeAIID = h.resolveActiveAIIDFromAIcall(ctx, ac)
		}
	}

	notice := PipelineErrorNotice{
		Type:          NotificationTypePipelineError,
		Category:      evt.Category,
		Fatal:         evt.Fatal,
		PipecatcallID: evt.PipecatcallID,
		Message:       text,
	}
	content, err := json.Marshal(notice)
	if err != nil {
		log.Errorf("Could not marshal the pipeline error notice. err: %v", err)
		return
	}

	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
		message.DirectionOutgoing, message.RoleNotification, string(content), nil, "",
		WithPipecatcallID(evt.PipecatcallID),
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the pipeline error notice. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created the pipeline error notice.")
}

// confirmForeignPipecatcall re-reads the aicall bypassing the cache when the cached row says the
// event's pipecatcall is foreign. The cache refresh on AIcallUpdate discards its own error, so a
// stale cached PipecatcallID could make the current turn look foreign (same guard as
// EventPMMessageBotLLM). The caller re-checks foreignness on the returned row. On a re-read failure
// this returns a copy of the cached row whose PipecatcallID is the event's, i.e. the event is
// treated as current and its notice is not windowed (fail-open; the bot-LLM path drops instead,
// but a missed error notice is worse than a duplicate).
func (h *messageHandler) confirmForeignPipecatcall(ctx context.Context, cached *aicall.AIcall, evtPipecatcallID uuid.UUID) *aicall.AIcall {
	fresh, err := h.reqHandler.AIV1AIcallGetSkipCache(ctx, cached.ID)
	if err != nil {
		logrus.WithField("aicall_id", cached.ID).Warnf("Could not re-read the aicall to confirm a foreign pipecatcall. Treating it as current. err: %v", err)
		tmp := *cached
		tmp.PipecatcallID = evtPipecatcallID
		return &tmp
	}
	return fresh
}

// pipelineErrorNoticeRecent reports whether the aicall already has a pipeline_error notice of the
// same category created within pipelineErrorNoticeWindow. Errors and unparseable rows count as no
// match (fail-open).
func (h *messageHandler) pipelineErrorNoticeRecent(ctx context.Context, aicallID uuid.UUID, category pmmessage.ErrorCategory) bool {
	filters := map[message.Field]any{
		message.FieldAIcallID: aicallID,
		message.FieldRole:     message.RoleNotification,
		message.FieldDeleted:  false,
	}
	msgs, err := h.db.MessageList(ctx, pipelineErrorNoticeScanSize, "", filters)
	if err != nil {
		logrus.WithField("aicall_id", aicallID).Warnf("Could not list notification messages for the dedup check. err: %v", err)
		return false
	}

	cutoff := h.utilHandler.TimeNow().Add(-pipelineErrorNoticeWindow)

	for _, m := range msgs {
		if m.TMCreate == nil || m.TMCreate.Before(cutoff) {
			continue
		}
		var n PipelineErrorNotice
		if errUnmarshal := json.Unmarshal([]byte(m.Content), &n); errUnmarshal != nil {
			continue
		}
		if n.Type == NotificationTypePipelineError && n.Category == category {
			return true
		}
	}
	return false
}
