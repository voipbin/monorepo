package arieventhandler

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
)

// EventHandlerRecordingStarted handles RecordingStarted ARI event
func (h *eventHandler) EventHandlerRecordingStarted(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.RecordingStarted)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerRecordingStarted",
		"event": e,
	})

	if !strings.HasSuffix(e.Recording.Name, "_in") {
		// for reference type call, we are making a 2 recordings channels for 1 recording
		// and it makes race condition. So, rather than process in/out both, we process only the "in"
		// because reference type confbridge, makes only "in".
		// so, nothing to do here if it's not "in".
		return nil
	}

	// parse recordingName and get recording
	recordingName := strings.TrimSuffix(e.Recording.Name, "_in")
	r, err := h.recordingHandler.GetByRecordingName(ctx, recordingName)
	if err != nil {
		log.Errorf("Could not get the recording. err: %v", err)
		return err
	}

	log = log.WithFields(logrus.Fields{
		"reference_type": r.ReferenceType,
		"reference_id":   r.ReferenceID,
	})
	log.WithField("recording", r).Debugf("Executing EventHandlerRecordingStarted event. recording_id: %s", r.ID)

	tmp, err := h.recordingHandler.Started(ctx, r.ID)
	if err != nil {
		log.Errorf("Could not handle the recording started. err: %v", err)
		return err
	}
	log.WithField("recording", tmp).Debugf("Updated recording status. recording_id: %s", tmp.ID)

	return nil
}

// EventHandlerRecordingFinished handles RecordingFinished ARI event
func (h *eventHandler) EventHandlerRecordingFinished(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.RecordingFinished)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerRecordingFinished",
		"event": e,
	})

	if !strings.HasSuffix(e.Recording.Name, "_in") {
		// for reference type call, we are making a 2 recordings channels for 1 recording
		// and it makes race condition. So, rather than process in/out both, we process only the "in"
		// because reference type confbridge, makes only "in".
		// so, nothing to do here if it's not "in".
		return nil
	}

	// parse recordingName and get recording
	recordingName := strings.TrimSuffix(e.Recording.Name, "_in")
	r, err := h.recordingHandler.GetByRecordingName(ctx, recordingName)
	if err != nil {
		log.Errorf("Could not get the recording. err: %v", err)
		return err
	}

	log = log.WithFields(
		logrus.Fields{
			"reference_type": r.ReferenceType,
			"reference_id":   r.ReferenceID,
		})
	log.WithField("recording", r).Debugf("Executing eventHandlerRecordingFinished event. recording_id: %s", r.ID)

	tmp, errStopped := h.recordingHandler.Stopped(ctx, r.ID)

	// the recording is over whatever Stopped returned (for example the file storing can fail),
	// so release the call's recording id regardless. A failure here is only logged: it must not
	// make the event be processed again.
	if r.ReferenceType == recording.ReferenceTypeCall {
		if errUnset := h.callHandler.UnsetRecordingID(ctx, r.ReferenceID, r.ID); errUnset != nil {
			log.Errorf("Could not unset the recording id of the call. err: %v", errUnset)
		}
	}

	if errStopped != nil {
		log.Errorf("Could not stopped the recording. err: %v", errStopped)
		return errStopped
	}
	log.WithField("recording", tmp).Debugf("Updated recording info. recording_info: %s", tmp.ID)

	return nil
}
