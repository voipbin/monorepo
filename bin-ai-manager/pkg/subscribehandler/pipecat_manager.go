package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	pmmessage "monorepo/bin-pipecat-manager/models/message"

	"github.com/sirupsen/logrus"
)

func (h *subscribeHandler) processEventPMMessageBotTranscription(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMMessageBotTranscription",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's message_bot_transcription event.")

	var evt pmmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.messageHandler.EventPMMessageBotTranscription(ctx, &evt)

	return nil
}

func (h *subscribeHandler) processEventPMMessageUserTranscription(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMMessageUserTranscription",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's message_user_transcription event.")

	var evt pmmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.messageHandler.EventPMMessageUserTranscription(ctx, &evt)

	return nil
}
