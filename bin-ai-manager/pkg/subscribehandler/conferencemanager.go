package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/sirupsen/logrus"
)

// processEventCMConferenceUpdated handles the conference-manager's call hangup event
func (h *subscribeHandler) processEventCMConferenceUpdated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventConferenceManager",
		"event": m,
	})

	evt := cfconference.Conference{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	go h.summaryHandler.EventCMConferenceUpdated(context.Background(), &evt)

	return nil
}
