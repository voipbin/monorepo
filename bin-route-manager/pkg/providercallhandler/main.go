package providercallhandler

//go:generate mockgen -package providercallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

type providerCallHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// ProviderCallHandler interface
type ProviderCallHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		providerID uuid.UUID,
		flowID uuid.UUID,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		anonymous string,
		callIDs []uuid.UUID,
		groupcallIDs []uuid.UUID,
	) (*providercall.ProviderCall, error)

	Get(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error)
	List(ctx context.Context, token string, limit uint64, filters map[providercall.Field]any) ([]*providercall.ProviderCall, error)
	Delete(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error)
}

// NewProviderCallHandler returns a ProviderCallHandler.
func NewProviderCallHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ProviderCallHandler {
	h := &providerCallHandler{
		db:            db,
		notifyHandler: notifyHandler,
	}

	return h
}
