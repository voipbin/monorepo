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

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// STTProvider represents a speech-to-text provider type
type STTProvider string

const (
	STTProviderGCP STTProvider = "GCP"
	STTProviderAWS STTProvider = "AWS"
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

	providerPriority []STTProvider // Validated list of providers in priority order

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

	// Parse and validate STT provider priority
	priorityList := strings.Split(config.Get().STTProviderPriority, ",")
	var validatedProviders []STTProvider

	for _, providerStr := range priorityList {
		provider, err := validateProvider(providerStr)
		if err != nil {
			log.Errorf("Could not validate STT provider. provider: %s, err: %v", providerStr, err)
			return nil
		}

		// Validate provider is initialized
		if provider == STTProviderGCP && gcpClient == nil {
			log.Errorf("STT provider '%s' listed in priority but not initialized (check GCP credentials)", STTProviderGCP)
			return nil
		}
		if provider == STTProviderAWS && awsClient == nil {
			log.Errorf("STT provider '%s' listed in priority but not initialized (check AWS credentials)", STTProviderAWS)
			return nil
		}

		validatedProviders = append(validatedProviders, provider)
	}

	if len(validatedProviders) == 0 {
		log.Error("No valid STT providers in priority list")
		return nil
	}

	// Convert to string slice for logging
	providerNames := make([]string, len(validatedProviders))
	for i, p := range validatedProviders {
		providerNames[i] = string(p)
	}
	log.Infof("STT provider priority: %s", strings.Join(providerNames, " → "))

	return &streamingHandler{
		utilHandler:       utilhandler.NewUtilHandler(),
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		transcriptHandler: transcriptHandler,

		listenAddress: listenAddress,

		gcpClient:        gcpClient,
		awsClient:        awsClient,
		providerPriority: validatedProviders,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muSteaming:   sync.Mutex{},
	}
}
