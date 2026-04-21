package providercallhandler

//go:generate mockgen -package providercallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

type providerCallHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// ProviderCallHandler interface
type ProviderCallHandler interface {
	// Create orchestrates an admin-triggered provider call end-to-end:
	// 1. If actions is non-empty and flowID is uuid.Nil, create a temp flow
	//    via FlowV1FlowCreate (cleaned up on failure).
	// 2. Build server-side metadata (route_provider_ids, skip_source_validation).
	// 3. Issue CallV1CallsCreate — call-manager persists the Call(s), forwards
	//    metadata to getDialroutes / getValidatedSourceForOutgoingCall.
	// 4. Persist the ProviderCall audit record with the resulting call/groupcall IDs.
	// 5. Publish the providercall_created event.
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		providerID uuid.UUID,
		flowID uuid.UUID,
		actions []fmaction.Action,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		anonymous string,
	) (*providercall.ProviderCall, error)

	Get(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error)
	List(ctx context.Context, token string, limit uint64, filters map[providercall.Field]any) ([]*providercall.ProviderCall, error)
	Delete(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error)
}

// NewProviderCallHandler returns a ProviderCallHandler.
func NewProviderCallHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ProviderCallHandler {
	h := &providerCallHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
