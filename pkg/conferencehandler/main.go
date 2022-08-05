package conferencehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package conferencehandler -destination ./mock_conferencehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	Create(
		ctx context.Context,
		conferenceType conference.Type,
		customerID uuid.UUID,
		name string,
		detail string,
		timeout int,
		preActions []action.Action,
		postActions []action.Action,
	) (*conference.Conference, error)
	Get(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	GetByConfbridgeID(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	Gets(ctx context.Context, customerID uuid.UUID, confType conference.Type, size uint64, token string) ([]*conference.Conference, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		timeout int,
		preActions []action.Action,
		postActions []action.Action,
	) (*conference.Conference, error)

	Join(ctx context.Context, conferenceID uuid.UUID, referenceType conferencecall.ReferenceType, referenceID uuid.UUID) (*conferencecall.Conferencecall, error)
	JoinedConfbridge(ctx context.Context, confbridgeID, callID uuid.UUID) error
	Leave(ctx context.Context, conferencecallID uuid.UUID) (*conferencecall.Conferencecall, error)
	Leaved(ctx context.Context, cf *conference.Conference, referenceID uuid.UUID) error
	Terminate(ctx context.Context, id uuid.UUID) error
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler

	conferencecallHandler conferencecallhandler.ConferencecallHandler
}

// List of default values
const (
	defaultDialTimeout = 60                           //nolint:deadcode,varcheck // default outgoing dial timeout
	defaultTimeStamp   = "9999-01-01 00:00:00.000000" // default timestamp
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
	cache cachehandler.CacheHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,

		conferencecallHandler: conferencecallHandler,
	}

	return h
}
