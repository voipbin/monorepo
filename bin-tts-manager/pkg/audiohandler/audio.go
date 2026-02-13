package audiohandler

import (
	"context"
	"fmt"
	"monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *audioHandler) AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, provider tts.Provider, voiceID string, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "audioCreate",
		"call_id": callID,
	})
	log.Debugf("Creating a new audio. lang: %s, provider: %s, voice_id: %s, filepath: %s", lang, provider, voiceID, filepath)

	switch provider {
	case tts.ProviderGCP:
		return h.gcpAudioCreate(ctx, callID, text, lang, voiceID, filepath)

	case tts.ProviderAWS:
		return h.awsAudioCreate(ctx, callID, text, lang, voiceID, filepath)

	case "":
		// no provider specified — try GCP first, fall back to AWS.
		// GCP is preferred because it auto-selects a voice when the language is not in our default map.
		gcpErr := h.gcpAudioCreate(ctx, callID, text, lang, voiceID, filepath)
		if gcpErr == nil {
			return nil
		}
		log.WithError(gcpErr).Warn("GCP audio provider failed, trying AWS")

		awsErr := h.awsAudioCreate(ctx, callID, text, lang, voiceID, filepath)
		if awsErr == nil {
			return nil
		}
		log.WithError(awsErr).Warn("AWS audio provider failed")

		log.Error("All audio providers failed")
		return fmt.Errorf("all audio providers failed. gcp: %v, aws: %v", gcpErr, awsErr)

	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}
