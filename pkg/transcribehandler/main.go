package transcribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcribehandler -destination ./mock_transcribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"golang.org/x/text/language"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/sttgoogle"
)

// TranscribeHandler is interface for service handle
type TranscribeHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		referenceID uuid.UUID,
		transType transcribe.Type,
		language string,
		direction common.Direction,
		transcripts []transcript.Transcript,
	) (*transcribe.Transcribe, error)
	Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)

	CallRecording(ctx context.Context, customerID, callID uuid.UUID, language string) ([]*transcribe.Transcribe, error)

	Recording(ctx context.Context, customerID uuid.UUID, recordingID uuid.UUID, language string) (*transcribe.Transcribe, error)

	StreamingTranscribeStart(ctx context.Context, customerID uuid.UUID, referenceID uuid.UUID, transType transcribe.Type, language string) (*transcribe.Transcribe, error)
	StreamingTranscribeStop(ctx context.Context, id uuid.UUID) error
}

// transcribeHandler structure for service handle
type transcribeHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	hostID    uuid.UUID
	sttGoogle sttgoogle.STTGoogle

	transcribeStreamingsMap map[uuid.UUID][]*streaming.Streaming
	transcribeStreamingsMu  sync.Mutex
}

// prometheus
var (
	metricsNamespace = "transcribe_manager"

	promNumberCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "transcribe_create_total",
			Help:      "Total number of created transcribe type.",
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
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	credentialPath string,
	hostID uuid.UUID,
) TranscribeHandler {

	sttGoogle := sttgoogle.NewSTTGoogle(reqHandler, db, notifyHandler, credentialPath)

	h := &transcribeHandler{
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,

		hostID:    hostID,
		sttGoogle: sttGoogle,

		transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
	}

	return h
}

// getBCP47LanguageCode returns BCP47 type of language code
func getBCP47LanguageCode(lang string) string {
	res := language.BCP47.Make(lang)

	if res.String() == "und" {
		// failed. use the default
		return "en-US"
	}

	return res.String()
}
