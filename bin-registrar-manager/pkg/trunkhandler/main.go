package trunkhandler

//go:generate mockgen -package trunkhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"regexp"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// TrunkHandler is interface for service handle
type TrunkHandler interface {
	CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		domainName string,
		authTypes []sipauth.AuthType,
		username string,
		password string,
		allowedIPs []string,
	) (*trunk.Trunk, error)
	Get(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)
	List(ctx context.Context, token string, limit uint64, filters map[trunk.Field]any) ([]*trunk.Trunk, error)
	GetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error)
	Update(ctx context.Context, id uuid.UUID, fields map[trunk.Field]any) (*trunk.Trunk, error)
	Delete(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

// trunkHandler structure for service handle
type trunkHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler

	notifyHandler notifyhandler.NotifyHandler
}

var (
	metricsNamespace = "registrar_manager"

	promTrunkCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "trunk_create_total",
			Help:      "Total number of created trunk.",
		},
	)

	promTrunkDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "trunk_delete_total",
			Help:      "Total number of deleted trunk.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promTrunkCreateTotal,
		promTrunkDeleteTotal,
	)
}

// NewTrunkHandler returns new service handler
func NewTrunkHandler(r requesthandler.RequestHandler, dbBin dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) TrunkHandler {

	h := &trunkHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    r,
		db:            dbBin,
		notifyHandler: notifyHandler,
	}

	return h
}

// isValidDomainName returns true if the given domainName is valid domain name for the domain
func isValidDomainName(domainName string) bool {
	reg := regexp.MustCompile(`^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$`)

	return reg.MatchString(domainName)
}
