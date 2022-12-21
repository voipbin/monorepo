package transcirpthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcirpthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"math/rand"
	"net"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"google.golang.org/api/option"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
)

const (
	defaultListenPortMin = 10000
	defaultListenPortMax = 11000
)

var defaultListenIP string // listen ip address

// list of default external media channel options.
//nolint:deadcode,varcheck
const (
	externalMediaOptEncapsulation  = "rtp"
	externalMediaOptTransport      = "udp"
	externalMediaOptConnectionType = "client"
	externalMediaOptFormat         = "ulaw"
	externalMediaOptDirection      = "both"
)

const (
	defaultEncoding          = speechpb.RecognitionConfig_MULAW
	defaultSampleRate        = 8000
	defaultAudioChannelCount = 1
)

// transcriptHandler structure for streaming handler
type transcriptHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	clientSpeech *speech.Client
}

// prometheus
var (
	metricsNamespace = "transcribe_manager"

	promTranscriptCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "transcript_transcript_create_total",
			Help:      "Total number of created transcribe type.",
		},
		[]string{"type"},
	)
)

// TranscriptHandler defines
type TranscriptHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		transcribeID uuid.UUID,
		direction transcript.Direction,
		message string,
		tmTranscript string,
	) (*transcript.Transcript, error)

	Start(ctx context.Context, tr *transcribe.Transcribe, direction transcript.Direction) (*streaming.Streaming, error)
	Stop(ctx context.Context, st *streaming.Streaming) error

	Recording(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, recordingID uuid.UUID, language string) (*transcript.Transcript, error)
}

// NewTranscriptHandler returns sttgoogle interface
func NewTranscriptHandler(
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,

	credentialPath string,
) TranscriptHandler {

	// create client speech
	clientSpeech, err := speech.NewClient(context.Background(), option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	return &transcriptHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,
		clientSpeech:  clientSpeech,
	}
}

func init() {
	defaultListenIP = getListenIP()

	prometheus.MustRegister(
		promTranscriptCreateTotal,
	)
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
