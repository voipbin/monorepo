package transcripthandler

import (
	"context"

	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// processFromBucket transcribes from the bucket file
func (h *transcriptHandler) processFromBucket(ctx context.Context, mediaLink string, language string) (*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "processFromBucket",
		"media_link": mediaLink,
	})

	// Send the contents of the audio file with the encoding and
	// and sample rate information to be transcripted.
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:            8000,
			LanguageCode:               language,
			EnableWordTimeOffsets:      true,
			EnableAutomaticPunctuation: true,
			Model:                      "phone_call",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{
				Uri: mediaLink,
			},
		},
	}

	op, err := h.clientSpeech.LongRunningRecognize(ctx, req)
	if err != nil {
		log.Errorf("Could not init google stt. err: %v", err)
		return nil, err
	}

	// wait for result
	resp, err := op.Wait(ctx)
	if err != nil {
		log.Errorf("Could not get google stt result. err: %v", err)
		return nil, err
	}

	// Print the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			log.Debugf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}

	// create transcript
	message := ""
	if len(resp.Results) > 0 {
		message = resp.Results[0].Alternatives[0].Transcript
	}
	ts := "0000-00-00 00:00:00.00000"
	res := &transcript.Transcript{
		Direction:    transcript.DirectionBoth,
		Message:      message,
		TMTranscript: ts,
	}

	return res, nil
}
