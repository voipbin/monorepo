package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
)

// ProviderCallCreate orchestrates an admin-triggered outbound call through a
// specific provider. The handler:
//  1. validates caller permission (PermissionProjectSuperAdmin)
//  2. validates the supplied provider_id (via RouteV1ProviderGet)
//  3. builds metadata server-side — route_provider_ids (forces the provider)
//     and skip_source_validation (preserves the admin-supplied source verbatim)
//  4. creates the underlying Call(s)/Groupcall(s) via CallV1CallsCreate
//  5. persists a ProviderCall audit record via RouteV1ProviderCallCreate
//     capturing the admin's request + the resulting call/groupcall IDs
//  6. returns the ProviderCall.WebhookMessage
func (h *serviceHandler) ProviderCallCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	providerID uuid.UUID,
	flowID uuid.UUID,
	actions []fmaction.Action,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	anonymous string,
) (*rmprovidercall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderCallCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"provider_id": providerID,
		"flow_id":     flowID,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	// permission — project admin only
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// validate the provider exists before creating a Call record (otherwise
	// call-manager would create a Call that immediately fails at DialrouteList)
	if _, err := h.reqHandler.RouteV1ProviderGet(ctx, providerID); err != nil {
		log.Errorf("Could not find the provider. err: %v", err)
		return nil, fmt.Errorf("provider not found. err: %v", err)
	}

	// if the caller supplied inline actions, create a temp flow so
	// CallV1CallsCreate has a flow to execute after answer. Mirrors the
	// customer call-create path (see serviceHandler.CallCreate).
	targetFlowID := flowID
	tempFlowCreated := false
	if targetFlowID == uuid.Nil && len(actions) > 0 {
		f, errFlow := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow for providercall", actions, uuid.Nil, false)
		if errFlow != nil {
			log.Errorf("Could not create temp flow for providercall. err: %v", errFlow)
			return nil, errFlow
		}
		targetFlowID = f.ID
		tempFlowCreated = true
	}
	// Best-effort cleanup: if downstream CallV1CallsCreate or
	// RouteV1ProviderCallCreate fails after we created the temp flow,
	// delete it so the flow-manager isn't leaked "tmp" rows.
	var returnErr error
	defer func() {
		if tempFlowCreated && returnErr != nil {
			if _, delErr := h.FlowDelete(ctx, a, targetFlowID); delErr != nil {
				log.Errorf("Could not clean up orphaned temp flow after error. flow_id: %s, err: %v", targetFlowID, delErr)
			}
		}
	}()

	// build server-side metadata: force the provider + preserve admin source
	metadata := map[string]interface{}{
		string(call.MetadataKeyRouteProviderIDs):     []string{providerID.String()},
		string(call.MetadataKeySkipSourceValidation): true,
	}

	calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(
		ctx,
		a.CustomerID,
		targetFlowID,
		uuid.Nil, // master_call_id
		source,
		destinations,
		false, // early_execution
		false, // connect
		anonymous,
		metadata,
	)
	if err != nil {
		log.Errorf("Could not create calls for providercall. err: %v", err)
		returnErr = err
		return nil, err
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

	tmp, err := h.reqHandler.RouteV1ProviderCallCreate(
		ctx,
		a.CustomerID,
		providerID,
		targetFlowID,
		source,
		destinations,
		anonymous,
		callIDs,
		groupcallIDs,
	)
	if err != nil {
		// Calls were created but providercall record was not. Admin can still
		// retrieve the Calls via GET /calls. v1 accepts this trade-off rather
		// than compensating with a Call delete.
		log.Errorf("Could not persist providercall record (calls were created: %v, groupcalls: %v). err: %v", callIDs, groupcallIDs, err)
		returnErr = err
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ProviderCallGet returns a single providercall.
func (h *serviceHandler) ProviderCallGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovidercall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ProviderCallGet",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"providercall_id": id,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.RouteV1ProviderCallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get providercall. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ProviderCallGets returns a paginated list of providercalls. Because the
// endpoint is gated by PermissionProjectSuperAdmin — a platform-level
// (cross-customer) role — the list is not scoped by customer_id. This
// matches the single-record Get/Delete methods, which also do not apply
// a customer boundary (per the PRD: "admin sees/controls every provider").
func (h *serviceHandler) ProviderCallGets(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, providerID uuid.UUID) ([]*rmprovidercall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderCallGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
		"provider_id": providerID,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	filters := map[rmprovidercall.Field]any{}
	if providerID != uuid.Nil {
		filters[rmprovidercall.FieldProviderID] = providerID
	}

	tmps, err := h.reqHandler.RouteV1ProviderCallGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get providercalls. err: %v", err)
		return nil, err
	}

	res := make([]*rmprovidercall.WebhookMessage, len(tmps))
	for i := range tmps {
		res[i] = tmps[i].ConvertWebhookMessage()
	}

	return res, nil
}

// ProviderCallDelete soft-deletes a providercall record and returns the deleted record.
func (h *serviceHandler) ProviderCallDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovidercall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ProviderCallDelete",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"providercall_id": id,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.RouteV1ProviderCallDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete providercall. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
