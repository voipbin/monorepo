package domainhandler

//go:generate mockgen -destination ./mock_domainhandler_domainhandler.go -package domainhandler -source ./main.go DomainHandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

// DomainHandler is interface for service handle
type DomainHandler interface {
	DomainCreate(ctx context.Context, d *models.Domain) (*models.Domain, error)
	DomainDelete(ctx context.Context, id uuid.UUID) error
	DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Domain, error)
	DomainUpdate(ctx context.Context, d *models.Domain) (*models.Domain, error)
}

const (
	constDomainSuffix = ".sip.voipbin.net"
)

// domainHandler structure for service handle
type domainHandler struct {
	reqHandler requesthandler.RequestHandler
	dbAst      dbhandler.DBHandler
	dbBin      dbhandler.DBHandler
	cache      cachehandler.CacheHandler
	extHandler extensionhandler.ExtensionHandler
}

var (
	metricsNamespace = "registrar_manager"

	// promCallCreateTotal = prometheus.NewCounterVec(
	// 	prometheus.CounterOpts{
	// 		Namespace: metricsNamespace,
	// 		Name:      "call_create_total",
	// 		Help:      "Total number of created call direction with type.",
	// 	},
	// 	[]string{"direction", "type"},
	// )

	// promCallHangupTotal = prometheus.NewCounterVec(
	// 	prometheus.CounterOpts{
	// 		Namespace: metricsNamespace,
	// 		Name:      "call_hangup_total",
	// 		Help:      "Total number of hungup call direction with type and reason.",
	// 	},
	// 	[]string{"direction", "type", "reason"},
	// )

	// promCallActionTotal = prometheus.NewCounterVec(
	// 	prometheus.CounterOpts{
	// 		Namespace: metricsNamespace,
	// 		Name:      "call_action_total",
	// 		Help:      "Total number of executed actions.",
	// 	},
	// 	[]string{"type"},
	// )
)

func init() {
	prometheus.MustRegister(
	// promCallCreateTotal,
	// promCallHangupTotal,
	// promCallActionTotal,
	)
}

// NewDomainHandler returns new service handler
func NewDomainHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, cache cachehandler.CacheHandler, extHandler extensionhandler.ExtensionHandler) DomainHandler {

	h := &domainHandler{
		reqHandler: r,
		dbAst:      dbAst,
		dbBin:      dbBin,
		cache:      cache,
		extHandler: extHandler,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
