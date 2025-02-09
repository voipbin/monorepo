package audiohandler

import (
	"context"
	"errors"
	"monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *audioHandler) AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "audioCreate",
		"call_id": callID,
	})
	log.Debugf("Creating a new audio. lang: %s, gender: %s, filepath: %s", lang, gender, filepath)

	handlers := []func(context.Context, uuid.UUID, string, string, tts.Gender, string) error{
		h.gcpAudioCreate,
		h.awsAudioCreate,
	}

	for _, handler := range handlers {
		if err := handler(ctx, callID, text, lang, gender, filepath); err == nil {
			return nil
		} else {
			log.WithError(err).Warn("Audio provider failed, trying next one")
		}
	}

	log.Error("All audio providers failed")
	return errors.New("all audio providers failed")
}
