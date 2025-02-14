package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"log"
	"sync"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go/service/transcribestreamingservice"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// StreamingHandler define
type StreamingHandler interface {
	Run() error

	Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error)

	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
}

// list of default external media channel options.
//
//nolint:deadcode,varcheck
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationAudioSocket)
	defaultTransport      = string(cmexternalmedia.TransportTCP)
	defaultConnectionType = "client"
	defaultFormat         = "ulaw"
)

// default gcp stt options
const (
	defaultGCPEncoding          = speechpb.RecognitionConfig_MULAW
	defaultGCPSampleRate        = 8000
	defaultGCPAudioChannelCount = 1
)

// default aws stt options
const (
	defaultAWSRegion            = "eu-central-1"
	defaultAWSEncoding          = transcribestreamingservice.MediaEncodingPcm
	defaultAWSSampleRate        = 8000
	defaultAWSAudioChannelCount = 1
)

type streamingHandler struct {
	utilHandler       utilhandler.UtilHandler
	reqHandler        requesthandler.RequestHandler
	notifyHandler     notifyhandler.NotifyHandler
	transcriptHandler transcripthandler.TranscriptHandler

	listenAddress string

	gcpClient *speech.Client

	awsClient *transcribestreaming.Client

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muSteaming   sync.Mutex
}

// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,

	listenAddress string,
	gcpCredentialBase64 string,
	awsAccessKey string,
	awsSecretKey string,

) StreamingHandler {

	decodedCredential, err := base64.StdEncoding.DecodeString(gcpCredentialBase64)
	if err != nil {
		log.Printf("Error decoding base64 credential: %v", err)
		return nil
	}

	// create gcp client
	gcpClient, err := speech.NewClient(context.Background(), option.WithCredentialsJSON(decodedCredential))
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	// create aws client
	awsClient, err := awsNewClient(awsAccessKey, awsSecretKey)
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	return &streamingHandler{
		utilHandler:       utilhandler.NewUtilHandler(),
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		transcriptHandler: transcriptHandler,

		listenAddress: listenAddress,

		gcpClient: gcpClient,
		awsClient: awsClient,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muSteaming:   sync.Mutex{},
	}
}
