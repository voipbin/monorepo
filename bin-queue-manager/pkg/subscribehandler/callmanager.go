package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processEventCMCallHangup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMCallHungup",
			"event": m,
		},
	)

	e := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.EventCallCallHangup(ctx, e.ID)

	return nil
}

// processEventCMConfbridgeJoined handles the call-manager's confbridge_joined event.
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoined",
			"event": m,
		},
	)

	e := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.EventCallConfbridgeJoined(ctx, e.JoinedCallID, e.Confbridge.ID)

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeLeaved",
			"event": m,
		},
	)

	e := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.EventCallConfbridgeLeaved(ctx, e.LeavedCallID, e.Confbridge.ID)

	return nil
}
