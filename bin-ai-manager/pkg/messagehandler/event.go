package messagehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/models/message"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

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

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
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

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message.")
}

func (h *messageHandler) EventPMMessageUserLLM(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageUserLLM",
		"event": evt,
	})

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
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

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "")
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")
}
