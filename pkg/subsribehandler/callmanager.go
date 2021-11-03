package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmnotifyhandler "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoinedLeaved(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	type ConfbridgeJoinedLeaved struct {
		ID           uuid.UUID `json:"id"`            // confbridge id
		ConferenceID uuid.UUID `json:"conference_id"` // conference id
		CallID       uuid.UUID `json:"call_id"`       // call id.
	}

	evt := ConfbridgeJoinedLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	switch m.Type {
	case string(cmnotifyhandler.EventTypeConfbridgeJoined):
		return h.conferenceHandler.Joined(ctx, evt.ConferenceID, evt.CallID)

	case string(cmnotifyhandler.EventTypeConfbridgeLeaved):
		return h.conferenceHandler.Leaved(ctx, evt.ConferenceID, evt.CallID)

	default:
		return fmt.Errorf("no handler found")
	}

}
