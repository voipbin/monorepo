package audiohandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package audiohandler -destination ./mock_audiohandler_audiohandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// AudioHandler intreface for audio handler
type AudioHandler interface {
	AudioCreate(ctx context.Context, callID uuid.UUID, text, lang, gender, filename string) error
}

type audioHandler struct {
	client *texttospeech.Client
}

// NewAudioHandler create AudioHandler
func NewAudioHandler(credentialPath string) AudioHandler {
	ctx := context.Background()

	// create client
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		client: client,
	}

	return h
}

// Init initialize the Audio handler
func (h *audioHandler) Init() {
}
