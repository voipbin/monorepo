package stthandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"

	"gitlab.com/voipbin/bin-manager/stt-manager.git/models/stt"
)

// Recording transcribe the recoring and send the webhook
func (h *sttHandler) Recording(recordingID uuid.UUID, language, webhookURI, webhookMethod string) error {

	// do stt recording
	tmp, err := h.transcribeRecording(recordingID, language)
	if err != nil {
		logrus.Errorf("Coudl not convert to text. err: %v", err)
		return err
	}

	s := &stt.STT{
		ID:            uuid.Must(uuid.NewV4()),
		Type:          stt.TypeRecording,
		ReferenceID:   recordingID,
		Language:      language,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
		Transcript:    tmp,
	}

	// send webhook
	go func() {
		if err := h.sendWebhook(s); err != nil {
			logrus.Errorf("Could not send the webhook correctly. err: %v", err)
		}
	}()

	return nil
}

// transcribeRecording transcribe the recording
func (h *sttHandler) transcribeRecording(recordingID uuid.UUID, language string) (string, error) {

	// validate language
	lang := h.getBCP47LanguageCode(language)

	// send a request to storage-handler to get a media link
	rec, err := h.reqHandler.SMRecordingGet(recordingID)
	if err != nil {
		return "", err
	}

	res, err := h.sttFromBucket(rec.BucketURI, lang)
	if err != nil {
		return "", err
	}

	return res, nil
}

// getBCP47LanguageCode returns BCP47 type of language code
func (h *sttHandler) getBCP47LanguageCode(lang string) string {
	tag := language.BCP47.Make(lang)

	if tag.String() == "und" {
		return "en-US"
	}

	return tag.String()
}
