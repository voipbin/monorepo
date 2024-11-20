package accounthandler

//go:generate mockgen -package accounthandler -destination ./mock_accounthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
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
