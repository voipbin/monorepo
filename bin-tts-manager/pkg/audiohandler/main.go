package audiohandler

//go:generate mockgen -package audiohandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/fs"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/internal/ttsconfig"
	"monorepo/bin-tts-manager/models/tts"
)

// AudioHandler intreface for audio handler
type AudioHandler interface {
	AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, provider tts.Provider, voiceID string, filename string) error
}

type audioHandler struct {
	gcpClient *texttospeech.Client

	// awsCreds hands the AWS key pair to the handler at the time of use. awsBuild builds a Polly
	// client from a pair (awsGetClient when nil). The client of the last pair is kept in a slot
	// and reused while the pair is unchanged; the keys themselves are read for every call.
	awsCreds ttsconfig.AWSCredentialsFunc
	awsBuild func(ctx context.Context, accessKey, secretKey string) (*polly.Client, error)

	mu         sync.Mutex
	slotAccess string
	slotSecret string
	slotClient *polly.Client
}

// list of default variables
const (
	defaultAudioEncoding texttospeechpb.AudioEncoding = texttospeechpb.AudioEncoding_LINEAR16
	defaultSampleRate    int32                        = 8000
	defaultChannelNum                                 = 1

	defaultFileMode fs.FileMode = 0644
)

// NewAudioHandler create AudioHandler
func NewAudioHandler(ctx context.Context, awsCreds ttsconfig.AWSCredentialsFunc, gcpEndpoint string) AudioHandler {
	log := logrus.WithField("func", "NewAudioHandler")

	gcpClient, err := gcpGetClient(ctx, gcpEndpoint)
	if err != nil {
		log.Errorf("Could not create a new gcp client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		gcpClient: gcpClient,
		awsCreds:  awsCreds,
		awsBuild:  awsGetClient,
	}

	return h
}
