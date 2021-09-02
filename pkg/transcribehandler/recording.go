package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// Recording transcribe the recoring and send the webhook
func (h *transcribeHandler) Recording(recordingID uuid.UUID, language string) (*transcribe.Transcribe, error) {

	// do transcribe recording
	tmp, err := h.transcribeRecording(recordingID, language)
	if err != nil {
		logrus.Errorf("Coudl not convert to text. err: %v", err)
		return nil, err
	}

	s := &transcribe.Transcribe{
		ID:            uuid.Must(uuid.NewV4()),
		Type:          transcribe.TypeRecording,
		ReferenceID:   recordingID,
		HostID:        h.hostID,
		Language:      language,
		WebhookURI:    "",
		WebhookMethod: "",
		Transcription: tmp,
	}

	return s, nil
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

// transcribeFromBucket does transcribe from the bucket file
func (h *transcribeHandler) transcribeFromBucket(mediaLink string, language string) (string, error) {

	ctx := context.Background()

	// Send the contents of the audio file with the encoding and
	// and sample rate information to be transcripted.
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 8000,
			LanguageCode:    language,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{
				Uri: mediaLink,
			},
		},
	}

	op, err := h.clientSpeech.LongRunningRecognize(ctx, req)
	if err != nil {
		return "", err
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return "", err
	}

	// Print the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			logrus.Debugf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}

	res := resp.Results[0].Alternatives[0].Transcript
	return res, nil
}
