package arieventhandler

import (
	"context"
	"fmt"

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

	filename := fmt.Sprintf("%s.%s", e.Recording.Name, e.Recording.Format)
	r, err := h.db.RecordingGetByFilename(ctx, filename)
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
	h.notifyHandler.PublishWebhookEvent(ctx, EventTypeRecordingStarted, c.WebhookURI, tmpRecording)

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

	filename := fmt.Sprintf("%s.%s", e.Recording.Name, e.Recording.Format)
	r, err := h.db.RecordingGetByFilename(ctx, filename)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return err
	}

	log = log.WithFields(
		logrus.Fields{
			"type":      r.Type,
			"reference": r.ReferenceID,
		})
	log.Debug("Executing eventHandlerRecordingFinished event.")

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

	h.notifyHandler.PublishWebhookEvent(ctx, EventTypeRecordingFinished, c.WebhookURI, tmpRecording)

	// set empty recordID
	switch r.Type {
	case recording.TypeCall:
		if err := h.db.CallSetRecordID(ctx, r.ReferenceID, uuid.Nil); err != nil {
			log.Errorf("Could not set call record id. err: %v", err)
		}

	case recording.TypeConference:
		if err := h.db.ConfbridgeSetRecordID(ctx, r.ReferenceID, uuid.Nil); err != nil {
			log.Errorf("Could not get conference record id. err: %v", err)
		}

	default:
		log.Errorf("Could not find correct tech type for recording. parse: %v", r.Type)
	}

	return nil
}
