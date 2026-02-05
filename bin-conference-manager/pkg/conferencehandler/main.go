package conferencehandler

//go:generate mockgen -package conferencehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmrecording "monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	Create(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		conferenceType conference.Type,
		name string,
		detail string,
		data map[string]interface{},
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*conference.Conference, error)
	Delete(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	Get(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	GetByConfbridgeID(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	List(ctx context.Context, size uint64, token string, filters map[conference.Field]any) ([]*conference.Conference, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		data map[string]interface{},
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*conference.Conference, error)
	UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*conference.Conference, error)
	AddConferencecallID(ctx context.Context, id uuid.UUID, conferencecallID uuid.UUID) (*conference.Conference, error)
	RemoveConferencecallID(ctx context.Context, cfID uuid.UUID, ccID uuid.UUID) (*conference.Conference, error)

	Terminating(ctx context.Context, id uuid.UUID) (*conference.Conference, error)

	RecordingStart(
		ctx context.Context,
		id uuid.UUID,
		activeflowID uuid.UUID,
		format cmrecording.Format,
		duration int,
		onEndFlowID uuid.UUID,
	) (*conference.Conference, error)
	RecordingStop(ctx context.Context, id uuid.UUID) (*conference.Conference, error)

	TranscribeStart(ctx context.Context, id uuid.UUID, lang string) (*conference.Conference, error)
	TranscribeStop(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	utilHandler   utilhandler.UtilHandler
}

// List of default values
const (
	defaultDialTimeout      = 60                           //nolint:deadcode,varcheck // default outgoing dial timeout
defaultRecordingTimeout = 86400                        // 24hours
)

var (
	metricsNamespace = "conference_manager"

	promConferenceCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_create_total",
			Help:      "Total number of created conference with type.",
		},
		[]string{"type"},
	)

	promConferenceCloseTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_close_total",
			Help:      "Total number of closed conference type.",
		},
		[]string{"type"},
	)

	promConferenceJoinTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_join_total",
			Help:      "Total number of joined calls to the conference with type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promConferenceCreateTotal,
		promConferenceCloseTotal,
		promConferenceJoinTotal,
	)
}

// NewConferenceHandler returns new service handler
func NewConferenceHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	return h
}
