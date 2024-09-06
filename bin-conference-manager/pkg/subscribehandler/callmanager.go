package subscribehandler

import (
	"context"
	"encoding/json"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConfbridgeJoined",
		"event": m,
	})

	evt := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get conferencecall
	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		// conferencecall not found. Not a conferencecall.
		return nil
	}

	_, err = h.conferencecallHandler.Joined(ctx, cc)
	if err != nil {
		log.Errorf("Could not join the conferencecall. conferencecall_id: %s, err: %v", cc.ID, err)
		return err
	}

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConfbridgeLeaved",
		"event": m,
	})

	evt := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get conferencecall
	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		// conferencecall not found. Not a conferencecall.
		return nil
	}

	_, err = h.conferencecallHandler.Terminated(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminated the conference call. err: %v", err)
		return err
	}

	return nil
}
