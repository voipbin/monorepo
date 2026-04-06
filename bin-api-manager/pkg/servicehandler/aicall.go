package servicehandler

import (
	"context"
	"fmt"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIcallCreate is a service handler for aicall creation.
func (h *serviceHandler) AIcallCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	assistanceType amaicall.AssistanceType,
	assistanceID uuid.UUID,
	referenceType amaicall.ReferenceType,
	referenceID uuid.UUID,
) (*amaicall.WebhookMessage, error) {

	// normalize "ai_team" to "team" — the direct resource_type uses "ai_team"
	// but the ai-manager only knows "team"
	if assistanceType == amaicall.AssistanceType(dmdirect.ResourceTypeAITeam) {
		assistanceType = amaicall.AssistanceTypeTeam
	}

	// resolve customer ID based on assistance type
	var customerID uuid.UUID
	switch assistanceType {
	case amaicall.AssistanceTypeAI:
		cb, err := h.aiGet(ctx, assistanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get ai info")
		}
		customerID = cb.CustomerID
	case amaicall.AssistanceTypeTeam:
		t, err := h.teamGet(ctx, assistanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get team info")
		}
		customerID = t.CustomerID
	default:
		return nil, fmt.Errorf("unsupported assistance type: %s", assistanceType)
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("user has no permission")
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("resource type not allowed")
		}
		if a.DirectScope.ResourceID != assistanceID {
			return nil, fmt.Errorf("resource not in scope")
		}
	}

	tmp, err := h.reqHandler.AIV1AIcallStart(
		ctx,
		assistanceType,
		assistanceID,
		uuid.Nil,
		referenceType,
		referenceID,
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
func (h *serviceHandler) AIcallGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amaicall.WebhookMessage, error) {

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("user has no permission")
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("resource type not allowed")
		}
	}

	// filters
	filters := map[string]string{
		"deleted":     "false", // we don't need deleted items
		"customer_id": a.CustomerID.String(),
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertAIcallFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert filters")
	}

	tmps, err := h.reqHandler.AIV1AIcallList(ctx, token, size, typedFilters)
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

// convertAIcallFilters converts map[string]string to map[amaicall.Field]any
func (h *serviceHandler) convertAIcallFilters(filters map[string]string) (map[amaicall.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amaicall.AIcall{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[amaicall.Field]any, len(typed))
	for k, v := range typed {
		result[amaicall.Field(k)] = v
	}

	return result, nil
}

// AIcallGet gets the aicall of the given id.
// It returns aicall if it succeed.
func (h *serviceHandler) AIcallGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	tmp, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("user has no permission")
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("resource type not allowed")
		}
		if tmp.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("resource not in scope")
		}
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIcallDelete deletes the aicall.
func (h *serviceHandler) AIcallDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	c, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, fmt.Errorf("user has no permission")
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("resource type not allowed")
		}
		if c.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("resource not in scope")
		}
	}

	tmp, err := h.reqHandler.AIV1AIcallDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
