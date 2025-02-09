package audiohandler

//go:generate mockgen -package audiohandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/fs"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/aws/aws-sdk-go/service/polly"
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
	awsClient *polly.Polly
}

// list of default variables
const (
	defaultAudioEncoding texttospeechpb.AudioEncoding = texttospeechpb.AudioEncoding_LINEAR16
	defaultSampleRate    int32                        = 8000
	defaultChannelNum                                 = 1

	defaultFileMode fs.FileMode = 0644
)

// NewAudioHandler create AudioHandler
func NewAudioHandler(ctx context.Context, awsAccessKey string, awsSecretKey string, gcpCredentialBase64 string) AudioHandler {
	log := logrus.WithField("func", "NewAudioHandler")

	gcpClient, err := gcpGetClient(ctx, gcpCredentialBase64)
	if err != nil {
		log.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	awsClient, err := awsGetClient(awsAccessKey, awsSecretKey)
	if err != nil {
		log.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		gcpClient: gcpClient,
		awsClient: awsClient,
	}

	return h
}
