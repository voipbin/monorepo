package arieventhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// EventHandlerRecordingStarted handles RecordingStarted ARI event
func (h *eventHandler) EventHandlerRecordingStarted(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.RecordingStarted)

	log := log.WithFields(
		log.Fields{
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"record":   e.Recording.Name,
		})

	if !strings.HasSuffix(e.Recording.Name, "_in") {
		// nothing to do here.
		return nil
	}

	// parse recordingName and get recording
	recordingName := strings.TrimSuffix(e.Recording.Name, "_in")
	r, err := h.db.RecordingGetByRecordingName(ctx, recordingName)
	if err != nil {
		log.Errorf("Could not get the recording. err: %v", err)
		return err
	}

	// update record state to recording
	if err := h.db.RecordingSetStatus(ctx, r.ID, recording.StatusRecording, string(e.Timestamp)); err != nil {
		log.Errorf("Could not update the recording status to recording. err: %v", err)
		return err
	}

	tmpRecording, err := h.db.RecordingGet(ctx, r.ID)
	if err != nil {
		log.Errorf("Could not get the updated recording info. err: %v", err)
		return err
	}

	// get call info
	c, err := h.db.CallGet(ctx, tmpRecording.ReferenceID)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, c.CustomerID, EventTypeRecordingStarted, tmpRecording)

	return nil
}

// EventHandlerRecordingFinished handles RecordingFinished ARI event
func (h *eventHandler) EventHandlerRecordingFinished(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.RecordingFinished)

	log := log.WithFields(
		log.Fields{
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"record":   e.Recording.Name,
			"func":     "eventHandlerRecordingFinished",
		})

	if !strings.HasSuffix(e.Recording.Name, "_in") {
		// nothing to do here.
		return nil
	}

	// parse recordingName and get recording
	recordingName := strings.TrimSuffix(e.Recording.Name, "_in")
	r, err := h.db.RecordingGetByRecordingName(ctx, recordingName)
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

	// update record state to end
	if err := h.db.RecordingSetStatus(ctx, r.ID, recording.StatusEnd, string(e.Timestamp)); err != nil {
		log.Errorf("Could not update the record status to end. err: %v", err)
		return err
	}

	tmpRecording, err := h.db.RecordingGet(ctx, r.ID)
	if err != nil {
		log.Errorf("Could not get the updated recording info. err: %v", err)
		return err
	}

	c, err := h.db.CallGet(ctx, r.ReferenceID)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return err
	}

	h.notifyHandler.PublishWebhookEvent(ctx, c.CustomerID, EventTypeRecordingFinished, tmpRecording)

	// set empty recordID
	switch r.ReferenceType {
	case recording.ReferenceTypeCall:
		if err := h.db.CallSetRecordingID(ctx, r.ReferenceID, uuid.Nil); err != nil {
			log.Errorf("Could not set call record id. err: %v", err)
		}

	case recording.ReferenceTypeConference:
		if err := h.db.ConfbridgeSetRecordID(ctx, r.ReferenceID, uuid.Nil); err != nil {
			log.Errorf("Could not get conference record id. err: %v", err)
		}

	default:
		log.Errorf("Could not find correct tech type for recording. parse: %v", r.ReferenceType)
	}

	return nil
}
