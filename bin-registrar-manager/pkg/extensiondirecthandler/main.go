package extensiondirecthandler

//go:generate mockgen -package extensiondirecthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// ExtensionDirectHandler is interface for extension direct handle
type ExtensionDirectHandler interface {
	Create(ctx context.Context, customerID, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	Delete(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	Get(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	GetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	GetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
	GetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
	Regenerate(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
}

// extensionDirectHandler structure for service handle
type extensionDirectHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler
}

// NewExtensionDirectHandler returns new handler
func NewExtensionDirectHandler(db dbhandler.DBHandler) ExtensionDirectHandler {
	h := &extensionDirectHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
	}
	return h
}
