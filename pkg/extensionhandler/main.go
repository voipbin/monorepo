package extensionhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package extensionhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

// ExtensionHandler is interface for service handle
type ExtensionHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		domainID uuid.UUID,
		ext string,
		password string,
	) (*extension.Extension, error)
	Delete(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	DeleteByDomainID(ctx context.Context, domainID uuid.UUID) ([]*extension.Extension, error)
	Get(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	GetByEndpoint(ctx context.Context, endpoint string) (*extension.Extension, error)
	GetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	Update(ctx context.Context, e *extension.Extension) (*extension.Extension, error)
}

// extensionHandler structure for service handle
type extensionHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	dbAst         dbhandler.DBHandler
	dbBin         dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

var (
	metricsNamespace = "registrar_manager"

	promExtensionCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "extension_create_total",
			Help:      "Total number of created extension.",
		},
	)

	promExtensionDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "extension_delete_total",
			Help:      "Total number of deleted extension.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promExtensionCreateTotal,
		promExtensionDeleteTotal,
	)
}

// NewExtensionHandler returns new service handler
func NewExtensionHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) ExtensionHandler {

	h := &extensionHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    r,
		dbAst:         dbAst,
		dbBin:         dbBin,
		notifyHandler: notifyHandler,
	}

	return h
}

func getStringPointer(v string) *string {
	return &v
}

func getIntegerPointer(v int) *int {
	return &v
}

// list of default variables
const (
	defaultMaxContacts    = 3          // default max registable contacts.
	defaultRemoveExisting = "yes"      // default remove existing.
	defaultAuthType       = "userpass" // default authentication method
)
