package customerhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package customerhandler -destination ./mock_customerhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// CustomerHandler interface
type CustomerHandler interface {
	CustomerCreate(ctx context.Context, username, password, name, detail, webhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*customer.Customer, error)
	CustomerDelete(ctx context.Context, id uuid.UUID) error
	CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	CustomerGets(ctx context.Context, size uint64, token string) ([]*customer.Customer, error)
	CustomerLogin(ctx context.Context, username, password string) (*customer.Customer, error)
	CustomerUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail, webhookMethod, webhookURI string) error
	CustomerUpdatePassword(ctx context.Context, id uuid.UUID, password string) error
	CustomerUpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) error
}

type customerHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewCustomerHandler return UserHandler interface
func NewCustomerHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) CustomerHandler {
	return &customerHandler{
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
