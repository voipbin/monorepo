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
	// ctx := context.Background()
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

	// get conference
	cf, err := h.db.ConferenceGetByConfbridgeID(ctx, evt.ID)
	if err != nil {
		// not found the conference
		log.Debugf("Could not get conference. err: %v", err)
		return nil
	}
	log = log.WithField("conference_id", cf.ID)
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if err := h.conferenceHandler.JoinedConfbridge(ctx, cf.ID, evt.JoinedCallID); err != nil {
		log.Errorf("Could not handle the confbridge joined event. err: %v", err)
		return err
	}

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	// ctx := context.Background()
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

	// get conference
	cf, err := h.db.ConferenceGetByConfbridgeID(ctx, evt.ID)
	if err != nil {
		// not found the conference
		log.Debugf("Could not get conference. err: %v", err)
		return nil
	}
	log = log.WithField("conference_id", cf.ID)
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if err := h.conferenceHandler.LeavedConfbridge(ctx, evt.ID, evt.LeavedCallID); err != nil {
		log.Errorf("Could not handle the confbridge leaved event. err: %v", err)
		return err
	}

	return nil
}
