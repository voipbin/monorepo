package messagehandler

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/sirupsen/logrus"
)

func (h *messageHandler) EventPMMessageBotTranscription(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageBotTranscription",
		"event": evt,
	})

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		return
	}

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message from the pipecat-manager's bot transcription.")
}

func (h *messageHandler) EventPMMessageUserTranscription(ctx context.Context, evt *pmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMMessageUserTranscription",
		"event": evt,
	})

	if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
		return
	}

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
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

	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created message from the pipecat-manager's user transcription.")
}
