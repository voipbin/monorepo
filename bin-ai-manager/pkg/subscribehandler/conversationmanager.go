package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/sirupsen/logrus"
)

// processEventCVMessageCreated handles conversation-manager's
// conversation_message_created event.
//
// It fires for EVERY conversation message on the platform (all channels, both
// directions) and processEventRun spawns a goroutine per event, so it does the
// minimum: unmarshal, hand off, return. The ownership filter (one Redis
// SMEMBERS) lives in aicallHandler.EventCVMessageCreated with the listen state.
func (h *subscribeHandler) processEventCVMessageCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCVMessageCreated",
		"event": m,
	})

	var evt cvmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventCVMessageCreated(ctx, &evt)

	return nil
}
