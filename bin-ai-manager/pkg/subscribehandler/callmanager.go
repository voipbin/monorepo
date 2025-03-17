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

	// get aicall
	cc, err := h.aicallHandler.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		// aicall not found. Not a aicall.
		return nil
	}

	_, err = h.aicallHandler.ProcessStart(ctx, cc)
	if err != nil {
		log.Errorf("Could not start the aicall. aicall_id: %s", err)
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

	// get aicall
	cc, err := h.aicallHandler.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		// aicall not found.
		return nil
	}

	_, err = h.aicallHandler.ProcessEnd(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminated the aicall call. err: %v", err)
		return err
	}

	return nil
}
