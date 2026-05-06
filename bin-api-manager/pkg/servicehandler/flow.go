package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// flowGet returns the flow info.
func (h *serviceHandler) flowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {

	res, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if res.TMDelete != nil {
		return nil, serviceerrors.ErrNotFound
	}

	return res, nil
}

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	detail string,
	actions []fmaction.Action,
	onCompleteID uuid.UUID,
	persist bool,
) (*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if persist {
		// we are making a persist flow here.
		// need to check if the customer has permission to create a persist flow.
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	} else {
		// we are making a non-persist flow here.
		// no need to check the agent's permission. checking the customer id is good enough
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	}

	if onCompleteID != uuid.Nil {
		// validate onCompleteID
		tmp, err := h.flowGet(ctx, onCompleteID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get the onComplete flow")
		}

		if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("%w: onComplete flow", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.FlowV1FlowCreate(ctx, a.CustomerID, fmflow.TypeFlow, name, detail, actions, onCompleteID, persist)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowDelete deletes the flow of the given id.
func (h *serviceHandler) FlowDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get flow
	f, err := h.flowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.FlowV1FlowDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowDirectHashRegenerate regenerates the direct hash for the flow.
func (h *serviceHandler) FlowDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, flowID uuid.UUID) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowDirectHashRegenerate",
		"customer_id": a.CustomerID,
		"flow_id":     flowID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Regenerating flow direct hash.")

	f, err := h.flowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not validate the flow info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.FlowV1FlowDirectHashRegenerate(ctx, flowID)
	if err != nil {
		log.Errorf("Could not regenerate flow direct hash. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get flow
	tmp, err := h.flowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGets gets the list of flow of the given customer id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
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
	flows, err := h.reqHandler.FlowV1FlowList(ctx, token, size, filters)
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
func (h *serviceHandler) FlowUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
	name string,
	detail string,
	actions []fmaction.Action,
	onCompleteID uuid.UUID,
) (*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get flows
	tmpFlow, err := h.flowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, tmpFlow.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if onCompleteID != uuid.Nil {
		tmp, err := h.flowGet(ctx, onCompleteID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get the onComplete flow")
		}

		if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("%w: onComplete flow", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdate(ctx, id, name, detail, actions, onCompleteID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowUpdateActions updates the flow's actions.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdateActions(ctx context.Context, a *auth.AuthIdentity, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get flows
	f, err := h.flowGet(ctx, flowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the flow")
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdateActions(ctx, flowID, actions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the flow")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
