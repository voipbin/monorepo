package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventTMTranscriptCreated handles the call-manager's call related event
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTMTranscriptCreated",
		"event": m,
	})

	var evt tmtranscript.Transcript
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	cb, err := h.aicallHandler.GetByTranscribeID(ctx, evt.TranscribeID)
	if err != nil {
		// no transcribe id found
		return nil
	}

	if errChat := h.aicallHandler.ChatMessage(ctx, cb, evt.Message); errChat != nil {
		log.Errorf("Could not chat to the aicall. err: %v", errChat)
		return errors.Wrap(errChat, "could not chat to the aicall")
	}

	return nil
}
