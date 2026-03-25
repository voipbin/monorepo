package directhandler

//go:generate mockgen -package directhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/cachehandler"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

// DirectHandler interfaces
type DirectHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, resourceType string, resourceID uuid.UUID) (*direct.Direct, error)
	Get(ctx context.Context, id uuid.UUID) (*direct.Direct, error)
	GetByHash(ctx context.Context, hash string) (*direct.Direct, error)
	Gets(ctx context.Context, size uint64, token string, filters map[direct.Field]any) ([]*direct.Direct, error)
	Delete(ctx context.Context, id uuid.UUID) (*direct.Direct, error)
	Regenerate(ctx context.Context, id uuid.UUID) (*direct.Direct, error)

	EventCustomerDeleted(ctx context.Context, customerID uuid.UUID) error
}

type directHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
	cache         cachehandler.CacheHandler
}

// NewDirectHandler return DirectHandler interface
func NewDirectHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, cacheHandler cachehandler.CacheHandler) DirectHandler {
	return &directHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
		cache:         cacheHandler,
	}
}
