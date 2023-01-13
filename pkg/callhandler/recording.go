package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// RecordingStart starts the call recording
func (h *callHandler) RecordingStart(
	ctx context.Context,
	id uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RecordingStart",
		"call_id": id,
	})

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	if c.RecordingID != uuid.Nil {
		log.Errorf("The call recording is already progressing. recording_id: %s", c.RecordingID)
		return fmt.Errorf("recording is already progressing")
	} else if c.Status != call.StatusProgressing {
		log.Errorf("The call is status is not progressing. status: %s", c.Status)
		return fmt.Errorf("invalid status")
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
		return err
	}
	log.WithField("recording", rec).Debugf("Started recording. recording_id: %s", rec.ID)

	return nil
}

// RecordingStop stops the recording
func (h *callHandler) RecordingStop(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RecordingStop",
		"call_id": id,
	})

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}
	log.WithField("conference", c).Debugf("Found conference info. conference_id: %s", c.ID)

	if c.RecordingID == uuid.Nil {
		log.Errorf("No Recording progressing. conference_id: %s, recording_id: %s", c.ID, c.RecordingID)
		return fmt.Errorf("no recording")
	} else if c.Status != call.StatusProgressing {
		log.Errorf("Invalid status. call_id: %s, status: %s", c.ID, c.Status)
		return fmt.Errorf("invalid status")
	}

	// send recording stop request
	tmp, err := h.recordingHandler.Stop(ctx, c.RecordingID)
	if err != nil {
		log.Errorf("Could not stop the recording. err: %v", err)
		return err
	}
	log.WithField("recording", tmp).Debugf("Recording is stopping. conference_id: %s, recording_id: %s", c.ID, tmp.ID)

	return nil
}
