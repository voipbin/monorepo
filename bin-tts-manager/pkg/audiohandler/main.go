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
func NewAudioHandler(credentialBase64 string) AudioHandler {
	log := logrus.WithField("func", "NewAudioHandler")

	ctx := context.Background()

	gcpClient, err := gcpGetClient(ctx, credentialBase64)
	if err != nil {
		log.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	// keepAliveParams := keepalive.ClientParameters{
	// 	Time:                30 * time.Second, // Ping every 30 seconds
	// 	Timeout:             10 * time.Second, // Wait 10 seconds for response
	// 	PermitWithoutStream: true,             // Send pings even if there are no active streams
	// }

	// decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	// if err != nil {
	// 	log.Printf("Error decoding base64 credential: %v", err)
	// 	return nil
	// }

	// // create client
	// client, err := texttospeech.NewClient(
	// 	ctx,
	// 	option.WithCredentialsJSON(decodedCredential),
	// 	option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
	// 	option.WithEndpoint("eu-texttospeech.googleapis.com:443"),
	// )
	// if err != nil {
	// 	logrus.Errorf("Could not create a new client. err: %v", err)
	// 	return nil
	// }

	h := &audioHandler{
		gcpClient: gcpClient,
	}

	return h
}
