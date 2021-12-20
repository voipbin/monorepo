package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmevent "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler/models/event"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoined handles the call-manager's confbridge_joined event.
func (h *subscribeHandler) processEventCMConfbridgeJoined(m *rabbitmqhandler.Event) error {
	// ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoined",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cmevent.ConfbridgeJoinedLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMConfbridgeLeaved(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cmevent.ConfbridgeJoinedLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Leaved(ctx, e.CallID, e.ID)

	return nil
}

// processEventCMCallHungup handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMCallHungup(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMCallHungup",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Hangup(ctx, e.ID)

	return nil
}
