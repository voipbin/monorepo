package accounthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package accounthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
)

const (
	maxFileSize = 1024 * 1024 * 1024 // 1GB per customer account
)

// AccountHandler intreface for GCP bucket handler
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*account.Account, error)
	Delete(ctx context.Context, id uuid.UUID) (*account.Account, error)

	IncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) (*account.Account, error)
	DecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) (*account.Account, error)

	ValidateFileInfoByCustomerID(ctx context.Context, customerID uuid.UUID, filecount int64, filesize int64) (*account.Account, error)

	EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer) error
	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
}

type accountHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

// NewAccountHandler create new account handler
func NewAccountHandler(
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) AccountHandler {

	h := &accountHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notifyHandler,
		db:            db,
	}

	return h
}
