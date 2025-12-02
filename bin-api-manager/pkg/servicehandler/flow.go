package servicehandler

import (
	"context"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// flowGet returns the flow info.
func (h *serviceHandler) flowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {

	res, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if res.TMDelete < defaultTimestamp {
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(ctx context.Context, a *amagent.Agent, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error) {

	if persist {
		// we are making a persist flow here.
		// need to check if the customer has permission to create a persist flow.
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("user has no permission")
		}
	} else {
		// we are making a non-persist flow here.
		// no need to check the agent's permission. checking the customer id is good enough
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
			return nil, fmt.Errorf("user has no permission")
		}
	}

	tmp, err := h.reqHandler.FlowV1FlowCreate(ctx, a.CustomerID, fmflow.TypeFlow, name, detail, actions, uuid.Nil, persist)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowDelete deletes the flow of the given id.
func (h *serviceHandler) FlowDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error) {

	// get flow
	f, err := h.flowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error) {

	// get flow
	tmp, err := h.flowGet(ctx, id)
	if err != nil {
		return nil, errors.Errorf("could not get the flow")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGets gets the list of flow of the given customer id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*fmflow.WebhookMessage, error) {

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[fmflow.Field]any{
		fmflow.FieldCustomerID: a.CustomerID,
		fmflow.FieldType:       fmflow.TypeFlow,
		fmflow.FieldDeleted:    false,
	}

	// get flows
	flows, err := h.reqHandler.FlowV1FlowGets(ctx, token, size, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get flows")
	}

	// create result
	res := []*fmflow.WebhookMessage{}
	for _, f := range flows {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// FlowUpdate updates the flow info.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdate(ctx context.Context, a *amagent.Agent, f *fmflow.Flow) (*fmflow.WebhookMessage, error) {

	// get flows
	tmpFlow, err := h.flowGet(ctx, f.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, tmpFlow.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdate(ctx, f)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowUpdateActions updates the flow's actions.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdateActions(ctx context.Context, a *amagent.Agent, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.WebhookMessage, error) {

	// get flows
	f, err := h.flowGet(ctx, flowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdateActions(ctx, flowID, actions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
