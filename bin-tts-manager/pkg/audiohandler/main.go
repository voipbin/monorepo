package audiohandler

//go:generate mockgen -package audiohandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/fs"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/tts"
)

// AudioHandler intreface for audio handler
type AudioHandler interface {
	AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filename string) error
}

type audioHandler struct {
	gcpClient *texttospeech.Client
}

// list of default variables
const (
	defaultAudioEncoding texttospeechpb.AudioEncoding = texttospeechpb.AudioEncoding_LINEAR16
	defaultSampleRate    int32                        = 8000

	defaultFileMode fs.FileMode = 0644
)

// NewAudioHandler create AudioHandler
func NewAudioHandler(ctx context.Context, credentialBase64 string) AudioHandler {
	log := logrus.WithField("func", "NewAudioHandler")

	gcpClient, err := gcpGetClient(ctx, credentialBase64)
	if err != nil {
		log.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		gcpClient: gcpClient,
	}

	return h
}
