package transcripthandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// processFromRecording transcribes from the bucket file
func (h *transcriptHandler) processFromRecording(ctx context.Context, mediaLink string, language string, direction transcript.Direction) ([]*transcript.Transcript, error) {
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
			// Model:                      "phone_call", // note: we can't use this model because it's not support for some languages(ex: ko-KR)
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

	res := []*transcript.Transcript{}
	var currentSentence string
	var sentenceStart time.Duration
	for _, result := range resp.Results {
		alt := result.Alternatives[0]
		for _, word := range alt.Words {
			if currentSentence == "" {
				sentenceStart = word.StartTime.AsDuration()
			}

			currentSentence += word.Word + " "
			if strings.HasSuffix(word.Word, ".") ||
				strings.HasSuffix(word.Word, "?") ||
				strings.HasSuffix(word.Word, "!") {

				res = append(res, &transcript.Transcript{
					Direction:    direction,
					Message:      strings.TrimSpace(currentSentence),
					TMTranscript: convertTime(sentenceStart),
				})

				currentSentence = ""
			}
		}
	}

	if currentSentence != "" {
		res = append(res, &transcript.Transcript{
			Direction:    direction,
			Message:      strings.TrimSpace(currentSentence),
			TMTranscript: convertTime(sentenceStart),
		})
	}

	return res, nil
}

func convertTime(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	microseconds := duration.Microseconds() % 1000000

	return fmt.Sprintf("0000-00-00 %02d:%02d:%02d.%05d", hours, minutes, seconds, microseconds)
}
