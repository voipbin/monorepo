package transcribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transcribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"golang.org/x/text/language"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/streaminghandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

// TranscribeHandler is interface for service handle
type TranscribeHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	GetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcribe.Transcribe, error)

	HealthCheck(ctx context.Context, id uuid.UUID, retryCount int)

	Start(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType transcribe.ReferenceType,
		referenceID uuid.UUID,
		language string,
		direction transcribe.Direction,
	) (*transcribe.Transcribe, error)
	Stop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
	EventCMCallHangup(ctx context.Context, c *cmcall.Call) error
	EventCMConfbridgeTerminated(ctx context.Context, c *cmconfbridge.Confbridge) error
}

// transcribeHandler structure for service handle
type transcribeHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	hostID            uuid.UUID
	transcriptHandler transcripthandler.TranscriptHandler
	streamingHandler  streaminghandler.StreamingHandler
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

// List of default variables
const (
	defaultHealthMaxRetryCount = 2
	defaultHealthDelay         = 10000 // 10 seconds
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
	streamingHandler streaminghandler.StreamingHandler,
	hostID uuid.UUID,
) TranscribeHandler {
	h := &transcribeHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,

		hostID:            hostID,
		transcriptHandler: transcriptHandler,
		streamingHandler:  streamingHandler,
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
