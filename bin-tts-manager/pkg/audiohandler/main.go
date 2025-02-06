package audiohandler

//go:generate mockgen -package audiohandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"io/fs"
	"log"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

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
func NewAudioHandler(credentialBase64 string) AudioHandler {
	ctx := context.Background()

	keepAliveParams := keepalive.ClientParameters{
		Time:                30 * time.Second, // Ping every 30 seconds
		Timeout:             10 * time.Second, // Wait 10 seconds for response
		PermitWithoutStream: true,             // Send pings even if there are no active streams
	}

	decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	if err != nil {
		log.Printf("Error decoding base64 credential: %v", err)
		return nil
	}

	// create client
	client, err := texttospeech.NewClient(
		ctx,
		option.WithCredentialsJSON(decodedCredential),
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
	)
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		client: client,
	}

	return h
}
