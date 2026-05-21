package messagehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	identity "monorepo/bin-common-handler/models/identity"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// deliveryStatusUpdateRetryDelay is the wait between the first and the (single) retried
// MessageUpdateDeliveryStatus call after a successful conversation send.
const deliveryStatusUpdateRetryDelay = 100 * time.Millisecond

// deliveryStatusUpdateSleep is a package-level indirection for time.Sleep so tests
// can patch it to avoid real wall-clock waits during the retry-path test case.
var deliveryStatusUpdateSleep = time.Sleep

// backstopGraceDelay is the grace window between receiving a pipecatcall_terminated
// event and re-checking whether an assistant reply has already been delivered. It
// gives any in-flight EventPMMessageBotLLM handler enough time to commit the row so
// the backstop short-circuits via MessageAssistantReplyExists.
const backstopGraceDelay = 3 * time.Second

// backstopReplyText is the user-visible fallback text the backstop persists and
// sends when no assistant reply was delivered before pipecatcall termination.
const backstopReplyText = "Sorry, I'm having trouble responding right now. Please try again."

// backstopGraceSleep is a package-level indirection for time.Sleep so tests can
// patch it to a no-op without paying real wall-clock time.
var backstopGraceSleep = time.Sleep

// resolveActiveAIIDFromAIcall returns the active AI UUID from an already-fetched AIcall.
// For AssistanceTypeAI it is ac.AssistanceID directly.
// For AssistanceTypeTeam it looks up the team and walks Members to find CurrentMemberID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveActiveAIIDFromAIcall(ctx context.Context, ac *aicall.AIcall) uuid.UUID {
	switch ac.AssistanceType {
	case aicall.AssistanceTypeAI:
		return ac.AssistanceID

	case aicall.AssistanceTypeTeam:
		t, err := h.db.TeamGet(ctx, ac.AssistanceID)
		if err != nil {
			logrus.Warnf("resolveActiveAIIDFromAIcall: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
			return uuid.Nil
		}
		for _, m := range t.Members {
			if m.ID == ac.CurrentMemberID {
				return m.AIID
			}
		}
		logrus.Warnf("resolveActiveAIIDFromAIcall: CurrentMemberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, ac.CurrentMemberID)
		return uuid.Nil

	default:
		logrus.Warnf("resolveActiveAIIDFromAIcall: unknown AssistanceType. type: %s", ac.AssistanceType)
		return uuid.Nil
	}
}

// resolveActiveAIID fetches the AIcall by ID, then delegates to resolveActiveAIIDFromAIcall.
// Use this at call sites that only have the aicall UUID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveActiveAIID(ctx context.Context, aicallID uuid.UUID) uuid.UUID {
	if h.reqHandler == nil {
		return uuid.Nil
	}
	ac, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
	if err != nil {
		logrus.Warnf("resolveActiveAIID: could not get aicall. aicall_id: %s, err: %v", aicallID, err)
		return uuid.Nil
	}
	return h.resolveActiveAIIDFromAIcall(ctx, ac)
}

// resolveTeamMemberAIID resolves the active AI UUID for a specific team member,
// independent of ac.CurrentMemberID. Used by EventPMTeamMemberSwitched where
// the notification message is created before UpdateCurrentMemberID commits.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveTeamMemberAIID(ctx context.Context, aicallID, memberID uuid.UUID) uuid.UUID {
	if h.reqHandler == nil {
		return uuid.Nil
	}
	ac, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
	if err != nil {
		logrus.Warnf("resolveTeamMemberAIID: could not get aicall. aicall_id: %s, err: %v", aicallID, err)
		return uuid.Nil
	}
	if ac.AssistanceType != aicall.AssistanceTypeTeam {
		return uuid.Nil
	}
	t, err := h.db.TeamGet(ctx, ac.AssistanceID)
	if err != nil {
		logrus.Warnf("resolveTeamMemberAIID: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
		return uuid.Nil
	}
	for _, m := range t.Members {
		if m.ID == memberID {
			return m.AIID
		}
	}
	logrus.Warnf("resolveTeamMemberAIID: memberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, memberID)
	return uuid.Nil
}

func (h *messageHandler) EventPMMessageUserTranscription(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageUserTranscription",
		"event": evt,
	})

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		return
	}

	activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "",
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message from the pipecat-manager's user transcription.")
}

func (h *messageHandler) EventPMMessageBotLLM(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageBotLLM",
		"event": evt,
	})

	if evt.Text == "" {
		// nothing to do
		return
	}

	// Only AIcall-typed pipecatcalls flow through the conversation-bridge logic.
	// All other pipecat reference types use the legacy "persist and return" path.
	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
			message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
		if err != nil {
			log.Errorf("Could not create the message. err: %v", err)
			return
		}
		log.WithField("message", tmp).Debugf("Created message.")
		return
	}

	ac, err := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
	if err != nil {
		log.WithField("aicall_id", evt.PipecatcallReferenceID).Errorf("Could not get aicall — skipping conversation delivery. err: %v", err)
		return
	}
	log.WithField("aicall", ac).Debugf("Retrieved aicall info. aicall_id: %s", ac.ID)

	// Voice / task: keep existing behavior — persist, no delivery.
	if ac.ReferenceType != aicall.ReferenceTypeConversation {
		activeAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
		tmp, errCreate := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
			message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
			WithActiveAIID(activeAIID))
		if errCreate != nil {
			log.Errorf("Could not create the message. err: %v", errCreate)
			return
		}
		log.WithField("message", tmp).Debugf("Created message.")
		return
	}

	// Guard #1 (primary) — drop stale responses BEFORE any DB write.
	// Per per-pod liveness preflight pattern: preflight before any DB write.
	if ac.PipecatcallID != evt.PipecatcallID {
		log.Infof("Dropping stale response (guard primary). aicall_id: %s, current_pcc: %s, event_pcc: %s",
			ac.ID, ac.PipecatcallID, evt.PipecatcallID)
		promConversationStaleResponseDroppedTotal.WithLabelValues("primary").Inc()
		return
	}

	// Persist the assistant message (only after guard #1 passes).
	// Mark delivery_status='pending' so a guard-#2 failure or send failure leaves the
	// row 'pending' and the periodic backstop can later finalize/cleanup the message.
	activeAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
	tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
		message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
		WithPipecatcallID(evt.PipecatcallID),
		WithDeliveryStatus(message.DeliveryStatusPending),
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message.")

	// Guard #2 (secondary) — re-check after persistence to narrow the dual-delivery race window.
	// On failure here the row stays 'pending' on purpose — the backstop will fire.
	acFinal, errFinal := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
	if errFinal != nil {
		log.WithField("aicall_id", evt.PipecatcallReferenceID).Warnf("Re-check AIcall fetch failed; skipping conversation delivery. err: %v", errFinal)
		return
	}
	if acFinal.PipecatcallID != evt.PipecatcallID {
		log.Infof("Race detected at delivery time (guard secondary). aicall_id: %s, event_pcc: %s",
			acFinal.ID, evt.PipecatcallID)
		promConversationStaleResponseDroppedTotal.WithLabelValues("secondary").Inc()
		return
	}

	// Deliver to conversation — silent failure on error per design.
	// NOTE: Do not add retry logic here. ConversationV1MessageSend dispatches to
	// SMS/LINE delivery downstream; a duplicate request would result in a duplicate
	// user-visible reply. If retry becomes necessary, gate it on an idempotency
	// key (e.g., evt.ID) honored by conversation-manager — see design doc §11
	// (Accepted v1 limits) for the dual-delivery race window.
	sent, errSend := h.reqHandler.ConversationV1MessageSend(ctx, acFinal.ReferenceID, evt.Text, []cvmedia.Media{})
	if errSend != nil {
		log.WithFields(logrus.Fields{
			"aicall_id":       acFinal.ID,
			"conversation_id": acFinal.ReferenceID,
			"event_id":        evt.ID,
		}).Errorf("Could not send conversation message (silent failure): %v", errSend)
		promConversationReplySendTotal.WithLabelValues("failure").Inc()
		// Row stays 'pending'; the backstop will fire.
		return
	}
	promConversationReplySendTotal.WithLabelValues("success").Inc()
	log.WithFields(logrus.Fields{
		"aicall_id":            acFinal.ID,
		"conversation_id":      acFinal.ReferenceID,
		"conversation_message": sent,
	}).Debugf("Sent conversation reply.")

	// Mark the row as delivered post-send. A single retry after a short delay is
	// allowed because the row is now committed and the send already succeeded;
	// the only remaining work is local DB bookkeeping. If both attempts fail,
	// we record the failure metric and rely on the backstop to reconcile.
	errUpd := h.db.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
	if errUpd != nil {
		deliveryStatusUpdateSleep(deliveryStatusUpdateRetryDelay)
		errUpd = h.db.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
	}
	if errUpd != nil {
		log.Errorf("Could not mark message delivered after retry. msg_id: %s err: %v", tmp.ID, errUpd)
		promConversationDeliveryStatusUpdateFailedTotal.Inc()
	}
}

func (h *messageHandler) EventPMMessageBotLLMIntermediate(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageBotLLMIntermediate",
		"event": evt,
	})

	if evt.Text == "" {
		return
	}

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		return
	}

	activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
	webhookMsg := &message.IntermediateWebhookMessage{
		Identity: identity.Identity{
			ID:         evt.ID,
			CustomerID: evt.CustomerID,
		},
		AIcallID:     evt.PipecatcallReferenceID,
		ActiveflowID: evt.ActiveflowID,
		ActiveAIID:   activeAIID,
		Role:         message.RoleAssistant,
		Content:      evt.Text,
		Direction:    message.DirectionIncoming,
		Sequence:     evt.Sequence,
	}

	h.notifyHandler.PublishWebhookEvent(ctx, evt.CustomerID, message.EventTypeMessageIntermediate, webhookMsg)
	log.Debugf("Published intermediate webhook event. message_id: %s, sequence: %d", evt.ID, evt.Sequence)
}

func (h *messageHandler) EventPMMessageUserLLM(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageUserLLM",
		"event": evt,
	})

	activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "",
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message.")
}

func (h *messageHandler) EventPMTeamMemberSwitched(ctx context.Context, evt *pmmessage.MemberSwitchedEvent) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMTeamMemberSwitched",
		"event": evt,
	})

	contentMap := map[string]any{
		"type":                     "member_switched",
		"transition_function_name": evt.TransitionFunctionName,
		"from_member": map[string]any{
			"id":   evt.FromMember.ID,
			"name": evt.FromMember.Name,
			"ai": map[string]any{
				"engine_model": evt.FromMember.EngineModel,
				"tts_type":     evt.FromMember.TTSType,
				"tts_voice_id": evt.FromMember.TTSVoiceID,
				"stt_type":     evt.FromMember.STTType,
			},
		},
		"to_member": map[string]any{
			"id":   evt.ToMember.ID,
			"name": evt.ToMember.Name,
			"ai": map[string]any{
				"engine_model": evt.ToMember.EngineModel,
				"tts_type":     evt.ToMember.TTSType,
				"tts_voice_id": evt.ToMember.TTSVoiceID,
				"stt_type":     evt.ToMember.STTType,
			},
		},
	}

	contentBytes, err := json.Marshal(contentMap)
	if err != nil {
		log.Errorf("Could not marshal notification content. err: %v", err)
		return
	}

	activeAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "",
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")
}

// EventPMPipecatcallTerminated is the backstop entrypoint for the
// pipecatcall_terminated event. It guarantees the AIcall+conversation branch
// always emits an assistant-side reply so the user does not see silence after
// the pipecatcall ends.
//
// Flow (per design doc §5):
//  1. Skip non-AICall pipecatcalls (label "skipped_not_aicall").
//  2. Resolve the AIcall. Skip non-conversation references (label "skipped_voice")
//     and AIcalls already terminated (label "skipped_terminated"). RPC errors
//     return nil (logged) so the subscribe handler does not requeue forever.
//  3. Sleep `backstopGraceDelay` to let any in-flight EventPMMessageBotLLM
//     handler commit its delivered row.
//  4. Re-check MessageAssistantReplyExists. If a delivered assistant reply
//     already exists for this pipecatcall, short-circuit (label "skipped_seen").
//  5. Persist the backstop message with delivery_status='delivered' BEFORE
//     calling ConversationV1MessageSend. This is intentional: a duplicate event
//     after retry will short-circuit at MessageAssistantReplyExists rather than
//     delivering a second reply downstream.
//  6. Send the backstop reply via the conversation manager. On send failure the
//     row is left in place (label "send_failed"); on persistence failure the
//     row never lands (label "failed").
func (h *messageHandler) EventPMPipecatcallTerminated(ctx context.Context, evt *pmpipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "EventPMPipecatcallTerminated",
		"pipecatcall_id": evt.ID,
	})

	if evt.ReferenceType != pmpipecatcall.ReferenceTypeAICall {
		promBackstopReplyTotal.WithLabelValues("skipped_not_aicall").Inc()
		return nil
	}

	ac, err := h.reqHandler.AIV1AIcallGet(ctx, evt.ReferenceID)
	if err != nil {
		log.Errorf("Could not get aicall. err: %v", err)
		return nil
	}

	if ac.ReferenceType != aicall.ReferenceTypeConversation {
		promBackstopReplyTotal.WithLabelValues("skipped_voice").Inc()
		return nil
	}
	if ac.Status == aicall.StatusTerminated {
		promBackstopReplyTotal.WithLabelValues("skipped_terminated").Inc()
		return nil
	}

	// Grace window — let any concurrent BotLLM handler commit its delivered row
	// so the next check short-circuits without persisting/sending a duplicate.
	backstopGraceSleep(backstopGraceDelay)

	seen, err := h.db.MessageAssistantReplyExists(ctx, evt.ID)
	if err != nil {
		return errors.Wrap(err, "could not check assistant reply existence")
	}
	if seen {
		promBackstopReplyTotal.WithLabelValues("skipped_seen").Inc()
		return nil
	}

	// Persist BEFORE sending. The row is created with delivery_status='delivered'
	// on purpose: a duplicated pipecatcall_terminated event after retry will see
	// the existing row via MessageAssistantReplyExists and short-circuit at the
	// "skipped_seen" branch above, preventing dual delivery.
	activeAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
	msg, err := h.Create(ctx, uuid.Nil, ac.CustomerID, ac.ID, ac.ActiveflowID,
		message.DirectionIncoming, message.RoleAssistant, backstopReplyText, nil, "",
		WithPipecatcallID(evt.ID),
		WithDeliveryStatus(message.DeliveryStatusDelivered),
		WithActiveAIID(activeAIID))
	if err != nil {
		promBackstopReplyTotal.WithLabelValues("failed").Inc()
		return errors.Wrap(err, "could not persist backstop message")
	}

	if _, errSend := h.reqHandler.ConversationV1MessageSend(ctx, ac.ReferenceID,
		backstopReplyText, []cvmedia.Media{}); errSend != nil {
		promBackstopReplyTotal.WithLabelValues("send_failed").Inc()
		return errors.Wrap(errSend, "could not send backstop conversation reply")
	}

	promBackstopReplyTotal.WithLabelValues("sent").Inc()
	log.WithField("message_id", msg.ID).Infof("Backstop reply sent. aicall_id: %s, pipecatcall_id: %s", ac.ID, evt.ID)
	return nil
}
