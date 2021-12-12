package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	cmnotifyhandler "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	cmnotifyhandlerevent "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler/models/event"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoinedLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoinedLeaved(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoinedLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	evt := cmnotifyhandlerevent.ConfbridgeJoinedLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	tmp, err := h.db.ConferenceConfbridgeGet(ctx, evt.ID)
	if err != nil {
		log.Errorf("Could not get conference-confbridge info. err: %v", err)
		return err
	}

	switch m.Type {
	case string(cmnotifyhandler.EventTypeConfbridgeJoined):
		return h.conferenceHandler.Joined(ctx, tmp.ConferenceID, evt.CallID)

	case string(cmnotifyhandler.EventTypeConfbridgeLeaved):
		return h.conferenceHandler.Leaved(ctx, tmp.ConferenceID, evt.CallID)

	default:
		return fmt.Errorf("no handler found")
	}

}
