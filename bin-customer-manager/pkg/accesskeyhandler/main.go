package accesskeyhandler

//go:generate mockgen -package accesskeyhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/dbhandler"
	"time"

	"github.com/gofrs/uuid"
)

const (
	defaultLenToken = 16
)

// AccesskeyHandler interface
type AccesskeyHandler interface {
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*accesskey.Accesskey, error)
	GetsByCustomerID(ctx context.Context, size uint64, token string, customerID uuid.UUID) ([]*accesskey.Accesskey, error)
	Get(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error)
	GetByToken(ctx context.Context, token string) (*accesskey.Accesskey, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		expire time.Duration,
	) (*accesskey.Accesskey, error)
	Delete(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error)
	UpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
	) (*accesskey.Accesskey, error)
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
