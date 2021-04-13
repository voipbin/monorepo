package transcribehandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// Recording transcribe the recoring and send the webhook
func (h *transcribeHandler) Recording(recordingID uuid.UUID, language, webhookURI, webhookMethod string) error {

	// do transcribe recording
	tmp, err := h.transcribeRecording(recordingID, language)
	if err != nil {
		logrus.Errorf("Coudl not convert to text. err: %v", err)
		return err
	}

	s := &transcribe.Transcribe{
		ID:            uuid.Must(uuid.NewV4()),
		Type:          transcribe.TypeRecording,
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
func (h *transcribeHandler) transcribeRecording(recordingID uuid.UUID, language string) (string, error) {

	// validate language
	lang := h.getBCP47LanguageCode(language)

	// send a request to storage-handler to get a media link
	rec, err := h.reqHandler.SMRecordingGet(recordingID)
	if err != nil {
		return "", err
	}

	res, err := h.transcribeFromBucket(rec.BucketURI, lang)
	if err != nil {
		return "", err
	}

	return res, nil
}

// getBCP47LanguageCode returns BCP47 type of language code
func (h *transcribeHandler) getBCP47LanguageCode(lang string) string {
	tag := language.BCP47.Make(lang)

	if tag.String() == "und" {
		return "en-US"
	}

	return tag.String()
}
