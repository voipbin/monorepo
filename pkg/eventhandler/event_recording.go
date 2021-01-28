package eventhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
)

// eventHandlerRecordingStarted handles RecordingStarted ARI event
func (h *eventHandler) eventHandlerRecordingStarted(ctx context.Context, evt interface{}) error {
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
		log.Errorf("Could not update the record status to recording. err: %v", err)
		return err
	}

	return nil
}

// eventHandlerRecordingFinished handles RecordingFinished ARI event
func (h *eventHandler) eventHandlerRecordingFinished(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.RecordingFinished)

	log := log.WithFields(
		log.Fields{
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"record":   e.Recording.Name,
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

	// set empty recordID
	switch r.Type {
	case recording.TypeCall:
		h.db.CallSetRecordID(ctx, r.ReferenceID, uuid.Nil)

	case recording.TypeConference:
		h.db.ConferenceSetRecordID(ctx, r.ReferenceID, uuid.Nil)

	default:
		log.Errorf("Could not find correct tech type for recording. parse: %v", r.Type)
	}

	return nil
}
