package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoined",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	evt := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	// get conferencecall
	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		// conferencecall not found. Not a conferencecall.
		return err
	}

	tmp, err := h.conferencecallHandler.Joined(ctx, cc)
	if err != nil {
		log.Errorf("Could not join the conferencecall. conferencecall_id: %s", err)
		return err
	}
	log.WithField("conferencecall", tmp).Debugf("Joined conferencecall. conferencecall_id: %s", tmp.ID)

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	evt := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	// get conferencecall
	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		// conferencecall not found. Not a conferencecall.
		return err
	}

	tmp, err := h.conferencecallHandler.Terminated(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminated the conference call. err: %v", err)
		return err
	}
	log.WithField("conferencecall", tmp).Debugf("Termianted conferencecall. conferencecall_id: %s", tmp.ID)

	return nil
}
