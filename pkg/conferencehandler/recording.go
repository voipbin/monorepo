package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// RecordingStart starts the conference recording
func (h *conferenceHandler) RecordingStart(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "RecordingStart",
		"conference_id": id,
	})

	cf, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if cf.RecordingID != uuid.Nil {
		log.Errorf("Recording is already progress. conference_id: %s, recording_id: %s", cf.ID, cf.RecordingID)
		return fmt.Errorf("already progress")
	} else if cf.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. conference_id: %s, status: %s", cf.ID, cf.Status)
		return fmt.Errorf("invalid status")
	}

	// send recording request
	tmp, err := h.reqHandler.CallV1RecordingStart(ctx, cmrecording.ReferenceTypeConference, cf.ID, "wav", 0, "", defaultRecordingTimeout)
	if err != nil {
		log.Errorf("Could not start the recording. err: %v", err)
		return err
	}
	log.WithField("recording", tmp).Debugf("Recording started. recording_id: %s", tmp.ID)

	return nil
}

// RecordingStop stops the recording
func (h *conferenceHandler) RecordingStop(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "RecordingStop",
		"conference_id": id,
	})

	cf, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if cf.RecordingID == uuid.Nil {
		log.Errorf("No Recording progressing. conference_id: %s, recording_id: %s", cf.ID, cf.RecordingID)
		return fmt.Errorf("no recording")
	} else if cf.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. conference_id: %s, status: %s", cf.ID, cf.Status)
		return fmt.Errorf("invalid status")
	}

	// send recording stop request
	tmp, err := h.reqHandler.CallV1RecordingStop(ctx, cf.RecordingID)
	if err != nil {
		log.Errorf("Could not stop the recording. err: %v", err)
		return err
	}
	log.WithField("recording", tmp).Debugf("Recording is stopping. conference_id: %s, recording_id: %s", cf.ID, tmp.ID)

	return nil
}
