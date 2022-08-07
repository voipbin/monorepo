package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// processEventConferenceConferencecallJoined handles the conference-manager's conferencecall_joined event.
func (h *subscribeHandler) processEventConferenceConferencecallJoined(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventConferenceConferencecallJoined",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cfconferencecall.Conferencecall{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Joined(ctx, e.ReferenceID, e.ConferenceID)

	return nil
}

// processEventConferenceConferencecallLeaved handles the conference-manager's conferencecall_leaved event.
func (h *subscribeHandler) processEventConferenceConferencecallLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventConferenceConferencecallLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := &cfconferencecall.Conferencecall{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Leaved(ctx, e.ReferenceID, e.ConferenceID)

	return nil
}
