package accounthandler

//go:generate mockgen -package accounthandler -destination ./mock_accounthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

// AccountHandler is interface for account handle
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID, providerData json.RawMessage) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	List(ctx context.Context, pageToken string, pageSize uint64, filters map[account.Field]any) ([]*account.Account, error)
	Update(ctx context.Context, id uuid.UUID, fields map[account.Field]any) (*account.Account, error)
	Delete(ctx context.Context, id uuid.UUID) (*account.Account, error)
}

// accountHandler structure for service handle
type accountHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler

	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	lineHandler    linehandler.LineHandler
	whatsappHandler whatsapphandler.WhatsAppHandler
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
	whatsappHandler whatsapphandler.WhatsAppHandler,
) AccountHandler {

	h := &accountHandler{
		utilHandler:     utilhandler.NewUtilHandler(),
		db:              db,
		reqHandler:      reqHandler,
		notifyHandler:   notifyHandler,
		lineHandler:     lineHandler,
		whatsappHandler: whatsappHandler,
	}

	return h
}
