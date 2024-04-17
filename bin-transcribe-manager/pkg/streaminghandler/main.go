package streaminghandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"math/rand"
	"net"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"google.golang.org/api/option"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

// StreamingHandler define
type StreamingHandler interface {
	Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error)
	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
}

// default variables
const (
	defaultListenPortMin = 10000
	defaultListenPortMax = 11000
)

var defaultListenIP string // listen ip address

// list of default external media channel options.
//nolint:deadcode,varcheck
const (
	constEncapsulation  = "rtp"
	constTransport      = "udp"
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
	db                dbhandler.DBHandler
	notifyHandler     notifyhandler.NotifyHandler
	transcriptHandler transcripthandler.TranscriptHandler

	clientSpeech *speech.Client
}

// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,

	credentialPath string,
) StreamingHandler {

	// create client speech
	clientSpeech, err := speech.NewClient(context.Background(), option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	return &streamingHandler{
		utilHandler:       utilhandler.NewUtilHandler(),
		reqHandler:        reqHandler,
		db:                db,
		notifyHandler:     notifyHandler,
		transcriptHandler: transcriptHandler,

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
	rand.Seed(time.Now().UTC().UnixNano())
	res := rand.Intn(defaultListenPortMax-defaultListenPortMin+1) + defaultListenPortMin
	return res
}
