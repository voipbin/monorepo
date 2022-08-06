package conferencecallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package conferencecallhandler -destination ./mock_conferencecallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

// ConferencecallHandler is interface for conferencecall handle
type ConferencecallHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		conferenceID uuid.UUID,
		referenceType conferencecall.ReferenceType,
		referenceID uuid.UUID,
	) (*conferencecall.Conferencecall, error)
	Get(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error)
	UpdateStatusJoined(ctx context.Context, conferencecallID uuid.UUID) (*conferencecall.Conferencecall, error)
	UpdateStatusLeaving(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	UpdateStatusLeaved(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
}

// conferencecallHandler structure for service handle
type conferencecallHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
}

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
func NewConferencecallHandler(req requesthandler.RequestHandler, notify notifyhandler.NotifyHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) ConferencecallHandler {

	h := &conferencecallHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,
	}

	return h
}
