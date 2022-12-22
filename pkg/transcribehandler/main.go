package transcribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"golang.org/x/text/language"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

// TranscribeHandler is interface for service handle
type TranscribeHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType transcribe.ReferenceType,
		referenceID uuid.UUID,
		language string,
		direction transcribe.Direction,
	) (*transcribe.Transcribe, error)
	Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	GetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*transcribe.Transcribe, error)

	TranscribingStart(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType transcribe.ReferenceType,
		referenceID uuid.UUID,
		language string,
		direction transcribe.Direction,
	) (*transcribe.Transcribe, error)
	TranscribingStop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
}

// transcribeHandler structure for service handle
type transcribeHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	hostID            uuid.UUID
	transcriptHandler transcripthandler.TranscriptHandler

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
	transcriptHandler transcripthandler.TranscriptHandler,
	hostID uuid.UUID,
) TranscribeHandler {
	h := &transcribeHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,

		hostID:            hostID,
		transcriptHandler: transcriptHandler,

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
