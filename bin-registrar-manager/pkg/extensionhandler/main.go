package extensionhandler

//go:generate mockgen -package extensionhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// ExtensionHandler is interface for service handle
type ExtensionHandler interface {
	CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		ext string,
		password string,
	) (*extension.Extension, error)
	Delete(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	Get(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	List(ctx context.Context, token string, limit uint64, filters map[extension.Field]any) ([]*extension.Extension, error)
	GetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error)
	Update(ctx context.Context, id uuid.UUID, fields map[extension.Field]any) (*extension.Extension, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
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
