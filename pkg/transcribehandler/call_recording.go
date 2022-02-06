package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// CallRecording transcribe the call's recordings
func (h *transcribeHandler) CallRecording(ctx context.Context, customerID, callID uuid.UUID, language string) ([]*transcribe.Transcribe, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":        "CallRecording",
			"customer_id": customerID,
			"call_id":     callID,
		},
	)

	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	// get call info
	c, err := h.reqHandler.CMV1CallGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	res := []*transcribe.Transcribe{}
	for _, recordingID := range c.RecordingIDs {

		// do transcribe recording
		tmp, err := h.sttGoogle.Recording(ctx, recordingID, lang)
		if err != nil {
			log.Errorf("Could not transcribe the recording. err: %v", err)
			continue
		}
		transcripts := []transcript.Transcript{*tmp}

		// create transcribe
		tr, err := h.Create(ctx, customerID, recordingID, transcribe.TypeRecording, lang, common.DirectionBoth, transcripts)
		if err != nil {
			log.Errorf("Could not create the transcribe. err: %v", err)
			continue
		}
		log.WithField("transcribe", tr).Debugf("Created the transcribe. transcribe_id: %s", tr.ID)

		h.notifyHandler.PublishWebhookEvent(ctx, tr.CustomerID, transcribe.EventTypeTranscribeCreated, tr)

		res = append(res, tr)
	}

	return res, nil
}
