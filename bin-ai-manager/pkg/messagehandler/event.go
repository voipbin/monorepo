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

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *messageHandler) EventPMMessageUserTranscription(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageUserTranscription",
		"event": evt,
	})

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		return
	}

	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
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
		log.Errorf("Could not get aicall — skipping conversation delivery. err: %v", err)
		return
	}
	log.WithField("aicall", ac).Debugf("Retrieved aicall info. aicall_id: %s", ac.ID)

	// Voice / task: keep existing behavior — persist, no delivery.
	if ac.ReferenceType != aicall.ReferenceTypeConversation {
		tmp, errCreate := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
			message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
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
	tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
		message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message.")

	// Guard #2 (secondary) — re-check after persistence to narrow the dual-delivery race window.
	acFinal, errFinal := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
	if errFinal != nil {
		log.Warnf("Re-check AIcall fetch failed; skipping conversation delivery. err: %v", errFinal)
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
		log.Errorf("Could not send conversation message (silent failure): %v", errSend)
		promConversationReplySendTotal.WithLabelValues("failure").Inc()
		return
	}
	promConversationReplySendTotal.WithLabelValues("success").Inc()
	log.WithField("conversation_message", sent).Debugf("Sent conversation reply.")
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

	webhookMsg := &message.IntermediateWebhookMessage{
		Identity: identity.Identity{
			ID:         evt.ID,
			CustomerID: evt.CustomerID,
		},
		AIcallID:     evt.PipecatcallReferenceID,
		ActiveflowID: evt.ActiveflowID,
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

	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
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

	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "")
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")
}
