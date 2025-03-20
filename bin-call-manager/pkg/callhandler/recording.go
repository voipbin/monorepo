package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
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
	onEndFlowID uuid.UUID,
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
		return nil, fmt.Errorf("recording is already progressing. recording_id: %s", c.RecordingID)
	} else if c.Status != call.StatusProgressing {
		return nil, fmt.Errorf("the call is not progressing. status: %s", c.Status)
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
		onEndFlowID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the recording")
	}
	log.WithField("recording", rec).Debugf("Started recording. recording_id: %s", rec.ID)

	res, err := h.UpdateRecordingID(ctx, c.ID, rec.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update recording id. recording_id: %s", rec.ID)
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
		return nil, errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Debugf("Found call info. call_id: %s", c.ID)

	if c.RecordingID == uuid.Nil {
		return nil, errors.Wrapf(err, "no recording is progressing. call_id: %s", c.ID)
	} else if c.Status != call.StatusProgressing {
		return nil, fmt.Errorf("the call is not progressing. status: %s", c.Status)
	}

	// send recording stop request
	tmp, err := h.recordingHandler.Stop(ctx, c.RecordingID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the recording")
	}
	log.WithField("recording", tmp).Debugf("Recording is stopping. conference_id: %s, recording_id: %s", c.ID, tmp.ID)

	res, err := h.UpdateRecordingID(ctx, c.ID, uuid.Nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update recording id")
	}

	return res, nil
}
