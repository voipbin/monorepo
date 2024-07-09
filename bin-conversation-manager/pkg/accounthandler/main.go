package accounthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package accounthandler -destination ./mock_accounthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

// AccountHandler is interface for account handle
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	Gets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]*account.Account, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) (*account.Account, error)
	Delete(ctx context.Context, id uuid.UUID) (*account.Account, error)
}

// accountHandler structure for service handle
type accountHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler

	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	lineHandler linehandler.LineHandler
}

var (
	metricsNamespace = "conversation_manager"

	promAccountCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "account_create_total",
			Help:      "Total number of created account with type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promAccountCreateTotal,
	)
}

// NewAccountHandler returns new account handler
func NewAccountHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	lineHandler linehandler.LineHandler,
) AccountHandler {

	h := &accountHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		lineHandler:   lineHandler,
	}

	return h
}
