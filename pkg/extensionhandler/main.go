package extensionhandler

//go:generate mockgen -destination ./mock_extensionhandler_extensionhandler.go -package extensionhandler -source ./main.go ExtensionHandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

// ExtensionHandler is interface for service handle
type ExtensionHandler interface {
	ExtensionCreate(ctx context.Context, e *extension.Extension) (*extension.Extension, error)
	ExtensionDelete(ctx context.Context, id uuid.UUID) error
	ExtensionDeleteByDomainID(ctx context.Context, domainID uuid.UUID) error
	ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	ExtensionGetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	ExtensionUpdate(ctx context.Context, e *extension.Extension) (*extension.Extension, error)
}

// extensionHandler structure for service handle
type extensionHandler struct {
	reqHandler requesthandler.RequestHandler
	dbAst      dbhandler.DBHandler
	dbBin      dbhandler.DBHandler
	cache      cachehandler.CacheHandler
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

// NewExtensionHandler returns new service handler
func NewExtensionHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, cache cachehandler.CacheHandler) ExtensionHandler {

	h := &extensionHandler{
		reqHandler: r,
		dbAst:      dbAst,
		dbBin:      dbBin,
		cache:      cache,
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

func getStringPointer(v string) *string {
	return &v
}

func getIntegerPointer(v int) *int {
	return &v
}
