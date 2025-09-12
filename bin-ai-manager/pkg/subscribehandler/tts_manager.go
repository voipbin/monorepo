package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	tmmessage "monorepo/bin-tts-manager/models/message"

	"github.com/sirupsen/logrus"
)

// processEventTMPlayFinished handles the tts-manager's play finished event
func (h *subscribeHandler) processEventTMPlayFinished(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTMPlayFinished",
		"event": m,
	})

	var evt tmmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventTMPlayFinished(ctx, &evt)

	return nil
}
