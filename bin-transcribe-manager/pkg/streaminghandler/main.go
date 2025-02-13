package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
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

	StartUDP(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error)
	StartTCP(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error)

	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
}

// default variables
const (
	defaultListenPortMin = 10000
	defaultListenPortMax = 11000
)

var defaultListenIP string // listen ip address

// list of default external media channel options.
//
//nolint:deadcode,varcheck
const (
	// constEncapsulation  = "rtp"
	constEncapsulation  = string(cmexternalmedia.EncapsulationAudioSocket)
	constTransport      = string(cmexternalmedia.TransportTCP)
	constConnectionType = "client"
	constFormat         = "ulaw"
	// externalMediaOptDirection = "both"
)

// const gcp stt options
const (
	defaultEncoding          = speechpb.RecognitionConfig_MULAW
	defaultSampleRate        = 8000
	defaultAudioChannelCount = 1
)

type streamingHandler struct {
	utilHandler       utilhandler.UtilHandler
	reqHandler        requesthandler.RequestHandler
	notifyHandler     notifyhandler.NotifyHandler
	transcriptHandler transcripthandler.TranscriptHandler

	listenAddress string

	clientSpeech *speech.Client

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muSteaming   sync.Mutex
}

// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,

	listenAddress string,
	credentialBase64 string,
) StreamingHandler {

	decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	if err != nil {
		log.Printf("Error decoding base64 credential: %v", err)
		return nil
	}

	// create client speech
	clientSpeech, err := speech.NewClient(context.Background(), option.WithCredentialsJSON(decodedCredential))
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

		clientSpeech: clientSpeech,
	}
}

func init() {
	defaultListenIP = getListenIP()
}

// getListenIP retrurns current listen ip address.
func getListenIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logrus.Errorf("Could not connect to the internet. err: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getRandomPort returns random listen port
func getRandomPort() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	res := r.Intn(defaultListenPortMax-defaultListenPortMin+1) + defaultListenPortMin
	return res
}
