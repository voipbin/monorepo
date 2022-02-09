package domainhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package domainhandler -destination ./mock_domainhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
)

// DomainHandler is interface for service handle
type DomainHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, domainName, name, detail string) (*domain.Domain, error)
	Delete(ctx context.Context, id uuid.UUID) (*domain.Domain, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Domain, error)
	Gets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*domain.Domain, error)
	Update(ctx context.Context, id uuid.UUID, name, detail string) (*domain.Domain, error)
}

const (
	constDomainSuffix = ".sip.voipbin.net"
)

// domainHandler structure for service handle
type domainHandler struct {
	reqHandler requesthandler.RequestHandler
	dbAst      dbhandler.DBHandler
	dbBin      dbhandler.DBHandler
	extHandler extensionhandler.ExtensionHandler

	notifyHandler notifyhandler.NotifyHandler
}

var (
	metricsNamespace = "registrar_manager"

	promDomainCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "domain_create_total",
			Help:      "Total number of created domain.",
		},
	)

	promDomainDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "domain_delete_total",
			Help:      "Total number of deleted domain.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promDomainCreateTotal,
		promDomainDeleteTotal,
	)
}

// NewDomainHandler returns new service handler
func NewDomainHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, extHandler extensionhandler.ExtensionHandler) DomainHandler {

	h := &domainHandler{
		reqHandler:    r,
		dbAst:         dbAst,
		dbBin:         dbBin,
		notifyHandler: notifyHandler,
		extHandler:    extHandler,
	}

	return h
}
