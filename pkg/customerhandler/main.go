package customerhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package customerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// CustomerHandler interface
type CustomerHandler interface {
	Create(
		ctx context.Context,
		username string,
		password string,
		name string,
		detail string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
		permissionIDs []uuid.UUID,
	) (*customer.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Get(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Gets(ctx context.Context, size uint64, token string) ([]*customer.Customer, error)
	Login(ctx context.Context, username, password string) (*customer.Customer, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod customer.WebhookMethod, webhookURI string) (*customer.Customer, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*customer.Customer, error)
	UpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) (*customer.Customer, error)
	UpdateBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) (*customer.Customer, error)

	IsValidBalance(ctx context.Context, customerID uuid.UUID, billingType bmbilling.ReferenceType, country string) (bool, error)
}

type customerHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewCustomerHandler return UserHandler interface
func NewCustomerHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) CustomerHandler {
	return &customerHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
	}
}

// checkHash returns true if the given hashstring is correct
func checkHash(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}

// GenerateHash generates hash from auth
func generateHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
