package eventhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
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

	recordID := e.Recording.Name

	// update record state to recording
	if err := h.db.RecordingSetStatus(ctx, recordID, recording.StatusRecording, string(e.Timestamp)); err != nil {
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
	log.Debug("Executing eventHandlerRecordingFinished event.")

	recordID := e.Recording.Name

	// update record state to end
	if err := h.db.RecordingSetStatus(ctx, recordID, recording.StatusEnd, string(e.Timestamp)); err != nil {
		log.Errorf("Could not update the record status to end. err: %v", err)
		return err
	}

	// set empty recordID
	tmpParse := strings.Split(recordID, "_")
	switch tmpParse[0] {
	case string(recording.TypeCall):
		h.db.CallSetRecordID(ctx, uuid.FromStringOrNil(tmpParse[1]), "")

	case string(recording.TypeConference):
		h.db.ConferenceSetRecordID(ctx, uuid.FromStringOrNil(tmpParse[1]), "")

	default:
		log.Errorf("Could not find correct tech type for recording. parse: %v", tmpParse)
	}

	return nil
}
