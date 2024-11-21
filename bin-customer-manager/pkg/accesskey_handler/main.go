package accesskey_handler

import (
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

const (
	defaultLenToken = 16
)

// AccesskeyHandler interface
type AccesskeyHandler interface {
	// Create(
	// 	ctx context.Context,
	// 	name string,
	// 	detail string,
	// 	email string,
	// 	phoneNumber string,
	// 	address string,
	// 	webhookMethod customer.WebhookMethod,
	// 	webhookURI string,
	// ) (*customer.Customer, error)
	// Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	// Get(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	// Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*customer.Customer, error)
	// UpdateBasicInfo(
	// 	ctx context.Context,
	// 	id uuid.UUID,
	// 	name string,
	// 	detail string,
	// 	email string,
	// 	phoneNumber string,
	// 	address string,
	// 	webhookMethod customer.WebhookMethod,
	// 	webhookURI string,
	// ) (*customer.Customer, error)
}

type accesskeyHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewAccesskeyHandler return UserHandler interface
func NewAccesskeyHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) AccesskeyHandler {
	return &accesskeyHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}
