package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

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

// STTProvider represents a speech-to-text provider type
type STTProvider string

const (
	STTProviderGCP STTProvider = "GCP"
	STTProviderAWS STTProvider = "AWS"
)

// StreamingHandler define
type StreamingHandler interface {
	Run() error

	Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction, provider transcribe.Provider) (*streaming.Streaming, error)

	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
}

// list of default external media channel options.
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationNone)
	defaultTransport      = string(cmexternalmedia.TransportWebsocket)
	defaultConnectionType = "server"
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

// AWSKeyFunc hands the AWS credential to the streaming handler at the time of use. An empty pair
// with no error means "AWS is not available"; an error means that the credential could not be
// read. It is called for every Start and must keep nothing a caller could hold.
type AWSKeyFunc = func(ctx context.Context) (accessKey, secretKey string, err error)

type streamingHandler struct {
	utilHandler       utilhandler.UtilHandler
	reqHandler        requesthandler.RequestHandler
	notifyHandler     notifyhandler.NotifyHandler
	transcriptHandler transcripthandler.TranscriptHandler

	gcpClient *speech.Client

	// awsKeys resolves the AWS credential for every Start. awsClient is the client built from the
	// pair awsAccess and awsSecret; it is reused while the pair does not change and replaced when
	// it does. muAWS guards the three fields. A client already handed to a running stream is not
	// touched by a replacement. newAWSClient builds a client (awsNewClient; tests replace it).
	awsKeys      AWSKeyFunc
	newAWSClient func(accessKey, secretKey string) (*transcribestreaming.Client, error)
	muAWS        sync.Mutex
	awsClient    *transcribestreaming.Client
	awsAccess    string
	awsSecret    string

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muSteaming   sync.Mutex
}

// NewStreamingHandler define
//
// The GCP client is created here, once (it uses application default credentials). The AWS
// credential is read by awsKeys for every Start, so a deployment that has only the AWS pair
// is not left without a provider, and a changed pair is used by the next Start. The handler is always the real one: when no provider is available Start answers
// STT_NOT_CONFIGURED, and nothing is decided at start up.
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,
	awsKeys AWSKeyFunc,
) StreamingHandler {
	log := logrus.WithField("func", "NewStreamingHandler")

	// Try to create GCP client (ADC-based)
	gcpClient, err := speech.NewClient(context.Background())
	if err != nil {
		log.Warnf("GCP client initialization failed (credentials not available): %v", err)
		gcpClient = nil
	}

	res := &streamingHandler{
		utilHandler:       utilhandler.NewUtilHandler(),
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		transcriptHandler: transcriptHandler,

		gcpClient: gcpClient,

		awsKeys:      awsKeys,
		newAWSClient: awsNewClient,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muSteaming:   sync.Mutex{},
	}

	// Report what is available now. A read failure only warns: the credential is read again at
	// every Start.
	awsClient, errAWS := res.awsClientFor(context.Background())
	if errAWS != nil {
		log.Warnf("Could not read the AWS credential at start up. err: %v", errAWS)
	}

	if gcpClient != nil {
		log.Info("GCP STT provider initialized")
	}
	if awsClient != nil {
		log.Info("AWS STT provider initialized")
	}
	if gcpClient == nil && awsClient == nil {
		log.Error("No STT provider is available now. Streaming transcribe is refused until GCP or AWS credentials are configured.")
	}

	return res
}

// awsClientFor returns the AWS client for the credential that awsKeys gives now: nil with no error
// when AWS is not available, an error when the credential could not be read. The credential is
// read outside the lock. The client is reused while the pair is the one it was built from, and
// built again (under the lock) when the pair changed. A client that can not be built is "AWS not
// available" and is not cached.
func (h *streamingHandler) awsClientFor(ctx context.Context) (*transcribestreaming.Client, error) {
	if h.awsKeys == nil {
		return nil, nil
	}

	accessKey, secretKey, err := h.awsKeys(ctx)
	if err != nil {
		return nil, err
	}
	if accessKey == "" || secretKey == "" {
		return nil, nil
	}

	h.muAWS.Lock()
	defer h.muAWS.Unlock()

	// Two Starts that read different pairs around a key change can replace each other's cache entry
	// here. Each one still uses the client of the pair it read, and the next Start compares again
	// and builds the client of the pair that is read then, so nothing stays wrong.
	if h.awsClient != nil && h.awsAccess == accessKey && h.awsSecret == secretKey {
		return h.awsClient, nil
	}

	client, err := h.newAWSClient(accessKey, secretKey)
	if err != nil {
		logrus.WithField("func", "awsClientFor").Warnf("AWS client initialization failed: %v", err)
		return nil, nil
	}

	h.awsClient, h.awsAccess, h.awsSecret = client, accessKey, secretKey

	return client, nil
}
