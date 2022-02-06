package sttgoogle

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package sttgoogle -destination ./mock_sttgoogle.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"math/rand"
	"net"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
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

// streamingHandler structure for streaming handler
type streamingHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	clientSpeech *speech.Client
}

// prometheus
var (
	metricsNamespace = "transcribe_manager"

	promSttCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "sttgoogle_transcribe_create_total",
			Help:      "Total number of created transcribe type.",
		},
		[]string{"type"},
	)
)

// STTGoogle defines
type STTGoogle interface {
	Start(ctx context.Context, tr *transcribe.Transcribe, direction common.Direction) (*streaming.Streaming, error)
	Stop(ctx context.Context, st *streaming.Streaming) error

	Recording(ctx context.Context, recordingID uuid.UUID, language string) (*transcript.Transcript, error)
}

// NewSTTGoogle returns sttgoogle interface
func NewSTTGoogle(
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	credentialPath string,
) STTGoogle {

	// create client speech
	clientSpeech, err := speech.NewClient(context.Background(), option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	return &streamingHandler{
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,
		clientSpeech:  clientSpeech,
	}
}

func init() {
	defaultListenIP = getListenIP()

	prometheus.MustRegister(
		promSttCreateTotal,
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
