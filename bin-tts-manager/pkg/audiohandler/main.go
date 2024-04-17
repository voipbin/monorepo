package audiohandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package audiohandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/fs"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"monorepo/bin-tts-manager/models/tts"
)

// AudioHandler intreface for audio handler
type AudioHandler interface {
	AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filename string) error
}

type audioHandler struct {
	client *texttospeech.Client
}

// list of default variables
const (
	defaultAudioEncoding texttospeechpb.AudioEncoding = texttospeechpb.AudioEncoding_LINEAR16
	defaultSampleRate    int32                        = 8000

	defaultFileMode fs.FileMode = 0644
)

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
