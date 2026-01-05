package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"sync"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
	defaultFormat         = "slin" // 8kHz, 16bit, mono signed linear PCM
)

// default gcp stt options
const (
	defaultGCPEncoding          = speechpb.RecognitionConfig_LINEAR16
	defaultGCPSampleRate        = 8000
	defaultGCPAudioChannelCount = 1
)

// default aws stt options
const (
	defaultAWSRegion     = "eu-central-1"
	defaultAWSEncoding   = types.MediaEncodingPcm
	defaultAWSSampleRate = 8000
)

const (
	defaultKeepAliveInterval = 10 * time.Second // 10 seconds
	defaultMaxRetryAttempts  = 3
	defaultInitialBackoff    = 100 * time.Millisecond // 100 milliseconds
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
	awsAccessKey string,
	awsSecretKey string,
) StreamingHandler {
	log := logrus.WithField("func", "NewStreamingHandler")

	// Try to create GCP client (ADC-based)
	gcpClient, err := speech.NewClient(context.Background())
	if err != nil {
		log.Warnf("GCP client initialization failed (credentials not available): %v", err)
		gcpClient = nil
	}

	// Only try AWS if credentials are provided
	var awsClient *transcribestreaming.Client
	if awsAccessKey != "" && awsSecretKey != "" {
		awsClient, err = awsNewClient(awsAccessKey, awsSecretKey)
		if err != nil {
			log.Warnf("AWS client initialization failed: %v", err)
			awsClient = nil
		}
	} else {
		log.Debug("AWS credentials not provided - AWS provider will be unavailable")
		awsClient = nil
	}

	// Validate at least one provider is available
	var providers []string
	if gcpClient != nil {
		providers = append(providers, "GCP")
	}
	if awsClient != nil {
		providers = append(providers, "AWS")
	}
	if len(providers) == 0 {
		log.Error("No STT providers available - at least one provider must be configured")
		return nil
	}
	log.Infof("STT providers initialized: %s", strings.Join(providers, ", "))

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
