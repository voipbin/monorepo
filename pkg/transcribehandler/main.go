package transcribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcribehandler -destination ./mock_transcribehandler_transcribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/requesthandler"
)

// TranscribeHandler is interface for service handle
type TranscribeHandler interface {
	CallRecording(callID uuid.UUID, language, webhookURI, webhookMethod string) error

	Recording(recordingID uuid.UUID, language string) (*transcribe.Transcribe, error)
}

// transcribeHandler structure for service handle
type transcribeHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	// gcp info
	clientStorgae *storage.Client
	clientSpeech  *speech.Client

	projectID  string
	bucketName string
	accessID   string
}

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

func init() {
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
