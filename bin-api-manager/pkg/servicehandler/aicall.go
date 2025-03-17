package servicehandler

import (
	"context"
	"fmt"

	amaicall "monorepo/bin-ai-manager/models/aicall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIcallCreate is a service handler for aicall creation.
func (h *serviceHandler) AIcallCreate(
	ctx context.Context,
	a *amagent.Agent,
	aiID uuid.UUID,
	referenceType amaicall.ReferenceType,
	referenceID uuid.UUID,
	gender amaicall.Gender,
	language string,
) (*amaicall.WebhookMessage, error) {

	cb, err := h.aiGet(ctx, aiID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai info")
	}

	if !h.hasPermission(ctx, a, cb.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1AIcallStart(
		ctx,
		aiID,
		referenceType,
		referenceID,
		gender,
		language,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// aicallGet returns the aicall info.
func (h *serviceHandler) aicallGet(ctx context.Context, id uuid.UUID) (*amaicall.AIcall, error) {
	// send request
	res, err := h.reqHandler.AIV1AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the resource info")
	}

	return res, nil
}

// AIcallGetsByCustomerID gets the list of aicalls of the given customer id.
// It returns list of AIs if it succeed.
func (h *serviceHandler) AIcallGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amaicall.WebhookMessage, error) {

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"deleted": "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.AIV1AIcallGetsByCustomerID(ctx, a.CustomerID, token, size, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicalls info")
	}

	// create result
	res := []*amaicall.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// AIcallGet gets the aicall of the given id.
// It returns aicall if it succeed.
func (h *serviceHandler) AIcallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	tmp, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIcallDelete deletes the aicall.
func (h *serviceHandler) AIcallDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	c, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1AIcallDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
