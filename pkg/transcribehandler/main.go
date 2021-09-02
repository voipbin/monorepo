package transcribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcribehandler -destination ./mock_transcribehandler_transcribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/rtphandler"
)

// TranscribeHandler is interface for service handle
type TranscribeHandler interface {
	CallRecording(callID uuid.UUID, language, webhookURI, webhookMethod string) error

	Recording(recordingID uuid.UUID, language string) (*transcribe.Transcribe, error)

	StreamingTranscribeStart(ctx context.Context, referenceID uuid.UUID, transType transcribe.Type, language string, webhookURI string, webhookMethod string) (*transcribe.Transcribe, error)
	StreamingTranscribeStop(ctx context.Context, id uuid.UUID) error

	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
}

// transcribeHandler structure for service handle
type transcribeHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	// gcp info
	clientStorgae *storage.Client
	clientSpeech  *speech.Client
	projectID     string
	bucketName    string
	accessID      string

	hostID uuid.UUID

	chanRawStream chan []byte
	servicePort   int

	rtpHandler rtphandler.RTPHandler
}

// prometheus
var (
	metricsNamespace = "transcribe_manager"

	promNumberCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "number_create_total",
			Help:      "Total number of created number type.",
		},
		[]string{"type"},
	)
)

const (
	defaultListenPortMin = 10000
	defaultListenPortMax = 11000
)

var defaultListenIP string // listen ip address

// streaming defines current streaming detail
type streaming struct {
	id     uuid.UUID
	conn   *net.UDPConn
	stream speechpb.Speech_StreamingRecognizeClient
}

// serviceStreamings contains currently serviced streaming list on the server
var serviceStreamings map[uuid.UUID]*streaming

func init() {
	defaultListenIP = getListenIP()

	prometheus.MustRegister(
		promNumberCreateTotal,
	)
}

// NewTranscribeHandler returns new service handler
func NewTranscribeHandler(
	r requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	credentialPath string,
	projectID string,
	bucketName string,
	hostID uuid.UUID,
) TranscribeHandler {

	ctx := context.Background()

	jsonKey, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		logrus.Errorf("Could not read the credential file. err: %v", err)
		return nil
	}

	// parse service account
	conf, err := google.JWTConfigFromJSON(jsonKey)
	if err != nil {
		logrus.Errorf("Could not parse the credential file. err: %v", err)
		return nil
	}

	// create client storage
	clientStorage, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client for storage. err: %v", err)
		return nil
	}

	// create client speech
	clientSpeech, err := speech.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client for speech. err: %v", err)
		return nil
	}

	h := &transcribeHandler{
		reqHandler: r,
		db:         db,
		cache:      cache,

		clientStorgae: clientStorage,
		clientSpeech:  clientSpeech,

		projectID:  projectID,
		bucketName: bucketName,
		accessID:   conf.Email,

		hostID: hostID,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Get preferred outbound ip of this machine
func getListenIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logrus.Errorf("Could not connect to the internet. err: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func getRandomPort() int {
	rand.Seed(time.Now().UTC().UnixNano())
	res := rand.Intn(defaultListenPortMax-defaultListenPortMin+1) + defaultListenPortMin
	return res
}

// getIPPort returns ip/port of the given net.Addr
func getIPPort(targetAddr net.Addr) (string, int) {
	var srcIP string
	var srcPort uint

	switch addr := targetAddr.(type) {
	case *net.UDPAddr:
		srcIP = addr.IP.String()
		srcPort = uint(addr.Port)
	case *net.TCPAddr:
		srcIP = addr.IP.String()
		srcPort = uint(addr.Port)
	}

	return srcIP, int(srcPort)
}

func addServiceStreaming(id uuid.UUID, conn *net.UDPConn, stream speechpb.Speech_StreamingRecognizeClient) {
	t := &streaming{
		id:     id,
		conn:   conn,
		stream: stream,
	}

	serviceStreamings[id] = t
}

func deleteServiceStreaming(id uuid.UUID) {
	delete(serviceStreamings, id)
}

func getServiceStreaming(id uuid.UUID) *streaming {
	if v, ok := serviceStreamings[id]; ok {
		return v
	}
	return nil
}
