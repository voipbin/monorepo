package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

func (h *conferenceHandler) Record(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Record",
		"conference_id": id,
	})

	conf, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}
	log.WithField("conference", conf).Debugf("Found conference info. conference_id: %s", conf.ID)

	if conf.RecordingID != uuid.Nil {
		log.Errorf("Recording is already progress. conference_id: %s, recording_id: %s", conf.ID, conf.RecordingID)
		return nil, fmt.Errorf("already progress")
	} else if conf.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. conference_id: %s, status: %s", conf.ID, conf.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// send recording request
	// h.reqHandler.Conf

	return nil, nil
}
