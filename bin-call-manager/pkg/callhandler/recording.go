package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
)

// RecordingStart starts the call recording
func (h *callHandler) RecordingStart(
	ctx context.Context,
	id uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RecordingStart",
		"call_id": id,
	})

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.RecordingID != uuid.Nil {
		log.Errorf("The call recording is already progressing. recording_id: %s", c.RecordingID)
		return nil, fmt.Errorf("recording is already progressing")
	} else if c.Status != call.StatusProgressing {
		log.Errorf("The call is status is not progressing. status: %s", c.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// starts the recording
	rec, err := h.recordingHandler.Start(
		ctx,
		recording.ReferenceTypeCall,
		c.ID,
		format,
		endOfSilence,
		endOfKey,
		duration,
	)
	if err != nil {
		log.Errorf("Could not start the recording. err: %v", err)
		return nil, err
	}
	log.WithField("recording", rec).Debugf("Started recording. recording_id: %s", rec.ID)

	res, err := h.UpdateRecordingID(ctx, c.ID, rec.ID)
	if err != nil {
		log.Errorf("Could not update recording id. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RecordingStop stops the recording
func (h *callHandler) RecordingStop(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RecordingStop",
		"call_id": id,
	})

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}
	log.WithField("conference", c).Debugf("Found conference info. conference_id: %s", c.ID)

	if c.RecordingID == uuid.Nil {
		log.Errorf("No Recording progressing. conference_id: %s, recording_id: %s", c.ID, c.RecordingID)
		return nil, fmt.Errorf("no recording")
	} else if c.Status != call.StatusProgressing {
		log.Errorf("Invalid status. call_id: %s, status: %s", c.ID, c.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// send recording stop request
	tmp, err := h.recordingHandler.Stop(ctx, c.RecordingID)
	if err != nil {
		log.Errorf("Could not stop the recording. err: %v", err)
		return nil, err
	}
	log.WithField("recording", tmp).Debugf("Recording is stopping. conference_id: %s, recording_id: %s", c.ID, tmp.ID)

	res, err := h.UpdateRecordingID(ctx, c.ID, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update recording id. err: %v", err)
		return nil, err
	}

	return res, nil
}
