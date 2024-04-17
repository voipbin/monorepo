package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventTMTranscriptCreated handles the call-manager's call related event
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTMTranscriptCreated",
		"event": m,
	})

	var evt tmtranscript.Transcript
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	cb, errChat := h.chatbotcallHandler.GetByTranscribeID(ctx, evt.TranscribeID)
	if errChat != nil {
		// no transcribe id found
		return nil
	}

	if errChat = h.chatbotcallHandler.ChatMessage(ctx, cb, evt.Message); errChat != nil {
		log.Errorf("Could not chat to the chatbotcall. err: %v", errChat)
		return errors.Wrap(errChat, "could not chat to the chatbotcall")
	}

	return nil
}
