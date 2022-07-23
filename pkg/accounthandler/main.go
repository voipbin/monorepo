package accounthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package accounthandler -destination ./mock_accounthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

// AccountHandler is interface for account handle
type AccountHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	Update(ctx context.Context, m *account.Account) error
	UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*account.Account, error)
}

// accountHandler structure for service handle
type accountHandler struct {
	db dbhandler.DBHandler

	reqHandler requesthandler.RequestHandler
}

// NewAccountHandler returns new account handler
func NewAccountHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler) AccountHandler {

	h := &accountHandler{
		db:         db,
		reqHandler: reqHandler,
	}

	return h
}
