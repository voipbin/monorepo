package ttshandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package ttshandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/models/tts"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/audiohandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/buckethandler"
)

// TTSHandler intreface for tts handler
type TTSHandler interface {
	Create(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender) (*tts.TTS, error)
}

type ttsHandler struct {
	credentailPath string

	audioHandler  audiohandler.AudioHandler
	bucketHandler buckethandler.BucketHandler
}

// NewTTSHandler create TTSHandler
func NewTTSHandler(credentialPath string, projectID, bucketName string) TTSHandler {
	audioHandler := audiohandler.NewAudioHandler(credentialPath)
	if audioHandler == nil {
		logrus.Errorf("Could not create audio handler.")
		return nil
	}

	bucketHandler := buckethandler.NewBucketHandler(credentialPath, projectID, bucketName)
	if bucketHandler == nil {
		logrus.Errorf("Could not create bucket handler.")
		return nil
	}

	h := &ttsHandler{
		credentailPath: credentialPath,

		audioHandler:  audioHandler,
		bucketHandler: bucketHandler,
	}

	return h
}
