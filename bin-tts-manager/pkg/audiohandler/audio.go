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

	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}
