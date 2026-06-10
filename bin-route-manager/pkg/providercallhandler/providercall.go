package providercallhandler

import (
	"context"
	stderrors "errors"

	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// Create orchestrates an admin-triggered provider call end-to-end inside
// bin-route-manager. See the interface doc on ProviderCallHandler.Create for
// the full sequence.
func (h *providerCallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	providerID uuid.UUID,
	flowID uuid.UUID,
	actions []fmaction.Action,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	anonymous string,
) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"provider_id": providerID,
	})

	// 1. If inline actions supplied without a flow_id, create a temp flow.
	//    The call-manager needs a concrete flow_id to attach to the outbound Call.
	targetFlowID := flowID
	tempFlowCreated := false
	if targetFlowID == uuid.Nil && len(actions) > 0 {
		f, errFlow := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "tmp", "tmp outbound flow for providercall", actions, uuid.Nil, false)
		if errFlow != nil {
			log.Errorf("Could not create temp flow for providercall. err: %v", errFlow)
			return nil, errors.Wrap(errFlow, "could not create temp flow")
		}
		targetFlowID = f.ID
		tempFlowCreated = true
	}

	// Best-effort cleanup of the temp flow when any subsequent step fails.
	// Uses a sentinel `returnErr` that every error-return path MUST set
	// before returning so the defer can tell success from failure.
	//
	// Known limitation: this defer is process-scoped. If route-manager
	// crashes (SIGKILL / OOM / pod eviction) between FlowV1FlowCreate and
	// the final return, the orphaned "tmp" flow is leaked — the runtime
	// never fires this defer. The admin-only, low-frequency nature of this
	// endpoint makes the accumulation negligible; if it ever becomes a
	// problem, a periodic flow-manager sweep of `name = "tmp"` + age >
	// threshold is the right mitigation, not a distributed transaction.
	var returnErr error
	defer func() {
		if tempFlowCreated && returnErr != nil {
			if _, delErr := h.reqHandler.FlowV1FlowDelete(ctx, targetFlowID); delErr != nil {
				log.Errorf("Could not clean up orphaned temp flow after error. flow_id: %s, err: %v", targetFlowID, delErr)
			}
		}
	}()

	// 2. Build server-side metadata — forces the provider via route-manager's
	//    synthetic dialroute and preserves the admin-supplied source verbatim.
	metadata := map[string]any{
		string(cmcall.MetadataKeyRouteProviderIDs):     []string{providerID.String()},
		string(cmcall.MetadataKeySkipSourceValidation): true,
		string(cmcall.MetadataKeyRTPDebug):             true, // force rtp_debug for all provider calls
	}

	// 3. Issue the call creation synchronously. Call-manager persists the Call(s),
	//    extracts route_provider_ids in getDialroutes → forwards to DialrouteList,
	//    honors skip_source_validation in getValidatedSourceForOutgoingCall.
	calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(
		ctx,
		customerID,
		targetFlowID,
		uuid.Nil, // master_call_id
		source,
		destinations,
		false, // early_execution
		false, // connect
		anonymous,
		metadata,
		nil,
	)
	if err != nil {
		log.Errorf("Could not create calls for providercall. err: %v", err)
		returnErr = err
		return nil, errors.Wrap(err, "could not create calls")
	}

	callIDs := make([]uuid.UUID, 0, len(calls))
	for _, c := range calls {
		if c != nil {
			callIDs = append(callIDs, c.ID)
		}
	}
	groupcallIDs := make([]uuid.UUID, 0, len(groupcalls))
	for _, g := range groupcalls {
		if g != nil {
			groupcallIDs = append(groupcallIDs, g.ID)
		}
	}

	// 4. Persist the ProviderCall audit record with the resulting IDs.
	id := uuid.Must(uuid.NewV4())
	p := &providercall.ProviderCall{
		ID:           id,
		CustomerID:   customerID,
		ProviderID:   providerID,
		FlowID:       targetFlowID,
		Source:       source,
		Destinations: destinations,
		Anonymous:    anonymous,
		CallIDs:      callIDs,
		GroupcallIDs: groupcallIDs,
	}
	log.WithField("providercall", p).Debugf("Persisting a new providercall. id: %s", id)

	if errCreate := h.db.ProviderCallCreate(ctx, p); errCreate != nil {
		// Calls were already created but we couldn't persist the audit record.
		// Admin can still retrieve the Calls via GET /v1/calls; v1 accepts this
		// trade-off rather than compensating with a Call delete.
		log.Errorf("Could not persist providercall record (calls created: %v, groupcalls: %v). err: %v", callIDs, groupcallIDs, errCreate)
		returnErr = errCreate
		return nil, errors.Wrap(errCreate, "could not persist providercall record")
	}

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created providercall info. err: %v", err)
		returnErr = err
		return nil, errors.Wrap(err, "could not get created providercall info")
	}
	h.notifyHandler.PublishEvent(ctx, providercall.EventTypeProviderCallCreated, res)

	return res, nil
}

// Get returns a single providercall by id.
func (h *providerCallHandler) Get(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get providercall. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameRouteManager,
				"PROVIDERCALL_NOT_FOUND",
				"The provider call was not found.",
			).Wrap(err)
		}
		return nil, err
	}
	log.WithField("providercall", res).Debugf("Retrieved providercall. id: %s", res.ID)

	return res, nil
}

// List returns a paginated list of providercalls.
func (h *providerCallHandler) List(ctx context.Context, token string, limit uint64, filters map[providercall.Field]any) ([]*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"token":   token,
		"limit":   limit,
		"filters": filters,
	})

	res, err := h.db.ProviderCallList(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get providercalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete soft-deletes the providercall and returns the deleted record.
func (h *providerCallHandler) Delete(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Delete",
		"providercall_id": id,
	})

	if err := h.db.ProviderCallDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the providercall. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the providercall")
	}

	res, err := h.db.ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted providercall. err: %v", err)
		return nil, errors.Wrap(err, "could not get deleted providercall")
	}
	h.notifyHandler.PublishEvent(ctx, providercall.EventTypeProviderCallDeleted, res)

	return res, nil
}
