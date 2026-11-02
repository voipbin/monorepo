package callhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
)

// recordingStartAnswerWait* bound how long RecordingStart waits for an answered call to
// reach the progressing status before starting the recording. They are variables only so
// that tests can shorten them.
var (
	recordingStartAnswerWaitTimeout  = 2 * time.Second
	recordingStartAnswerWaitInterval = 100 * time.Millisecond
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
	}

	// answer the call if it has not been answered yet (mirrors the Talk auto-answer
	// behavior). A hung-up call is rejected by channelHandler.Answer (tm_delete set), so a
	// terminated call still cannot start a recording.
	// The call status flips to progressing asynchronously via the ARI ChannelStateChange
	// event, and the recording handler rejects a call that is not progressing. So wait,
	// bounded, for the transition before starting the recording.
	if c.Status != call.StatusProgressing {
		log.Debugf("The call is not progressing yet. Answering the call before recording. status: %s", c.Status)
		if errAnswer := h.channelHandler.Answer(ctx, c.ChannelID); errAnswer != nil {
			return nil, errors.Wrap(errAnswer, "could not answer the call before recording")
		}

		if errWait := h.waitCallProgressing(ctx, c.ID); errWait != nil {
			return nil, errors.Wrap(errWait, "the answered call did not become progressing")
		}
	}

	// starts the recording
	rec, err := h.recordingHandler.Start(
		ctx,
		c.ActiveflowID,
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
		return nil, fmt.Errorf("no recording is progressing. call_id: %s", c.ID)
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

// waitCallProgressing polls the call until its status becomes progressing.
// It gives up after recordingStartAnswerWaitTimeout, when the call is hung up, or when the
// context is done.
func (h *callHandler) waitCallProgressing(ctx context.Context, id uuid.UUID) error {
	deadline := time.Now().Add(recordingStartAnswerWaitTimeout)
	for {
		c, err := h.Get(ctx, id)
		if err != nil {
			return errors.Wrapf(err, "could not get call info")
		}

		switch c.Status {
		case call.StatusProgressing:
			return nil
		case call.StatusHangup, call.StatusTerminating:
			return fmt.Errorf("the call is ending. status: %s", c.Status)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for the call to be progressing. status: %s", c.Status)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(recordingStartAnswerWaitInterval):
		}
	}
}
