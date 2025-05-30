package conferencecallhandler

//go:generate mockgen -package conferencecallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

// ConferencecallHandler is interface for conferencecall handle
type ConferencecallHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		conferenceID uuid.UUID,
		referenceType conferencecall.ReferenceType,
		referenceID uuid.UUID,
	) (*conferencecall.Conferencecall, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conferencecall.Conferencecall, error)
	Get(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error)

	Joined(ctx context.Context, cc *conferencecall.Conferencecall) (*conferencecall.Conferencecall, error)

	ServiceStart(
		ctx context.Context,
		activeflowID uuid.UUID,
		conferenceID uuid.UUID,
		referenceType conferencecall.ReferenceType,
		referenceID uuid.UUID,
	) (*commonservice.Service, error)

	Terminate(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	Terminated(ctx context.Context, cc *conferencecall.Conferencecall) (*conferencecall.Conferencecall, error)

	HealthCheck(ctx context.Context, id uuid.UUID, retryCount int)
}

// conferencecallHandler structure for service handle
type conferencecallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler

	conferenceHandler conferencehandler.ConferenceHandler
}

const (
	defaultHealthCheckDelay    = 5000 // 5 secs
	defaultHealthCheckRetryMax = 2    //

	maxConferencecallDuration = time.Hour * 24 // 1 day(24 hours)
)

var (
	metricsNamespace = "conference_manager"

	promConferencecallTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conferencecall_total",
			Help:      "Total number of conferencecall with reference_type and status.",
		},
		[]string{"reference_type", "status"},
	)
)

func init() {
	prometheus.MustRegister(
		promConferencecallTotal,
	)
}

// NewConferencecallHandler returns new service handler
func NewConferencecallHandler(req requesthandler.RequestHandler, notify notifyhandler.NotifyHandler, db dbhandler.DBHandler, conferenceHandler conferencehandler.ConferenceHandler) ConferencecallHandler {

	h := &conferencecallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,

		conferenceHandler: conferenceHandler,
	}

	return h
}
