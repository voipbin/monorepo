package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoined handles the call-manager's confbridge_joined event.
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoined",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Joined(ctx, e.JoinedCallID, e.ID)

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Leaved(ctx, e.LeavedCallID, e.ID)

	return nil
}

// processEventCMCallHungup handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMCallHungup(ctx context.Context, m *rabbitmqhandler.Event) error {
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

	h.queuecallHandler.Hungup(ctx, e.ID)

	return nil
}
