package customerhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package customerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// CustomerHandler interface
type CustomerHandler interface {
	Create(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
	) (*customer.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Get(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*customer.Customer, error)
	UpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
	) (*customer.Customer, error)
	UpdateBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) (*customer.Customer, error)

	IsValidBalance(ctx context.Context, customerID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error)
}

type customerHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewCustomerHandler return UserHandler interface
func NewCustomerHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) CustomerHandler {
	return &customerHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}
