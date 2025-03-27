package transcripthandler

//go:generate mockgen -package transcripthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"log"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
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
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcript.Transcript, error)
	Delete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error)

	Recording(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, recordingID uuid.UUID, language string) (*transcript.Transcript, error)
}

// NewTranscriptHandler returns sttgoogle interface
func NewTranscriptHandler(
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,

	credentialBase64 string,
) TranscriptHandler {

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

	return &transcriptHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,
		clientSpeech:  clientSpeech,
	}
}

func init() {
	prometheus.MustRegister(
		promTranscriptCreateTotal,
	)
}
