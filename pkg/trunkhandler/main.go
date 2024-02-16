package trunkhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package trunkhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

// TrunkHandler is interface for service handle
type TrunkHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		domainName string,
		authTypes []trunk.AuthType,
		username string,
		password string,
		allowedIPs []string,
	) (*trunk.Trunk, error)
	Get(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)
	GetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error)
	Gets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*trunk.Trunk, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, authTypes []trunk.AuthType, username string, password string, allowedIPs []string) (*trunk.Trunk, error)
	Delete(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)
}

// trunkHandler structure for service handle
type trunkHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler

	notifyHandler notifyhandler.NotifyHandler
}

var (
	basicDomainName  = "trunk.voipbin.net"
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
