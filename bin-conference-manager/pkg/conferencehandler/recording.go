package conferencehandler

import (
	"context"
	"fmt"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conference"
)

// RecordingStart starts the conference recording
func (h *conferenceHandler) RecordingStart(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "RecordingStart",
		"conference_id": id,
	})

	cf, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if cf.RecordingID != uuid.Nil {
		log.Errorf("Recording is already progress. conference_id: %s, recording_id: %s", cf.ID, cf.RecordingID)
		return nil, fmt.Errorf("already progress")
	} else if cf.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. conference_id: %s, status: %s", cf.ID, cf.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// send recording request
	tmp, err := h.reqHandler.CallV1RecordingStart(ctx, cmrecording.ReferenceTypeConfbridge, cf.ConfbridgeID, cmrecording.FormatWAV, 0, "", defaultRecordingTimeout)
	if err != nil {
		log.Errorf("Could not start the recording. err: %v", err)
		return nil, err
	}
	log.WithField("recording", tmp).Debugf("Recording started. recording_id: %s", tmp.ID)

	res, err := h.UpdateRecordingID(ctx, id, tmp.ID)
	if err != nil {
		log.Errorf("Could not update transcribe id. err: %v", err)
		return nil, err
	}
	log.WithField("conference", tmp).Debugf("Stopped transcribe. conference_id: %s, transcribe_id: %s", tmp.ID, res.RecordingID)

	return res, nil
}

// RecordingStop stops the recording
func (h *conferenceHandler) RecordingStop(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "RecordingStop",
		"conference_id": id,
	})

	cf, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	if cf.RecordingID == uuid.Nil {
		log.Errorf("No Recording progressing. conference_id: %s, recording_id: %s", cf.ID, cf.RecordingID)
		return nil, fmt.Errorf("no recording")
	} else if cf.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. conference_id: %s, status: %s", cf.ID, cf.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// send recording stop request
	tmp, err := h.reqHandler.CallV1RecordingStop(ctx, cf.RecordingID)
	if err != nil {
		log.Errorf("Could not stop the recording. err: %v", err)
		return nil, err
	}
	log.WithField("recording", tmp).Debugf("Recording is stopping. conference_id: %s, recording_id: %s", cf.ID, tmp.ID)

	res, err := h.UpdateRecordingID(ctx, id, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update transcribe id. err: %v", err)
		return nil, err
	}
	log.WithField("conference", res).Debugf("Stopped transcribe. conference_id: %s, transcribe_id: %s", res.ID, res.RecordingID)

	return res, nil
}
