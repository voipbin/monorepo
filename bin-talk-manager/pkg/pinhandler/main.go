package pinhandler

//go:generate mockgen -package pinhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/pin"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// defaultPageSize is the size of a page when the caller gives none.
const defaultPageSize uint64 = 100

// PinHandler defines the business logic of the personal pins of an owner.
type PinHandler interface {
	// PinCreate pins a chat or a message for the owner. It is idempotent: pinning a target again
	// returns the existing pin. The chat id of the pin is filled from the target.
	PinCreate(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, targetID uuid.UUID) (*pin.Pin, error)

	// PinList returns the pins of the owner that the owner can see, newest first.
	// targetType and chatID filter the result when they are set.
	PinList(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, chatID uuid.UUID, token string, size uint64) ([]*pin.Pin, error)

	// PinDelete deletes a pin of the owner and returns the deleted pin. A pin of another owner or
	// customer is reported as not found.
	PinDelete(ctx context.Context, id, customerID uuid.UUID, ownerType string, ownerID uuid.UUID) (*pin.Pin, error)
}

type pinHandler struct {
	dbHandler   dbhandler.DBHandler
	utilHandler commonutil.UtilHandler
}

// New creates a new PinHandler.
func New(dbHandler dbhandler.DBHandler, utilHandler commonutil.UtilHandler) PinHandler {
	return &pinHandler{
		dbHandler:   dbHandler,
		utilHandler: utilHandler,
	}
}
