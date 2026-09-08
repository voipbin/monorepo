package servicehandler

import (
	"context"
	"fmt"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	log := logrus.WithFields(logrus.Fields{
		"func":            "AIcallCreate",
		"assistance_type": assistanceType,
		"assistance_id":   assistanceID,
	})

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
		return nil, fmt.Errorf("%w: unsupported assistance type: %s", serviceerrors.ErrInvalidArgument, assistanceType)
	}

	// uuid.Nil lets ai-manager generate the id. Only the direct path pins it.
	aicallID := uuid.Nil

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
		}
		if a.DirectScope.ResourceID != assistanceID {
			return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
		}
		if a.DirectScope.AllowedResourceID == uuid.Nil {
			return nil, fmt.Errorf("%w: token is not resource bound", serviceerrors.ErrPermissionDenied)
		}

		// The new aicall must land on the id the token is bound to, or nothing
		// downstream can tell this visitor's resource from any other's.
		aicallID = a.DirectScope.AllowedResourceID

		// Force both reference fields. reference_type=none routes to
		// startReferenceTypeNone, the one start path that never calls
		// AIcallGetByReferenceID -- which has no customer filter and would
		// otherwise hand back another tenant's aicall (VOIP-1502).
		// reference_id=Nil is separately load-bearing: the generated column
		// active_reference_key keys off reference_id being non-zero, not off
		// the type, so forcing only the type would leave the unique-index
		// collision surface live. Neither forcing is redundant.
		//
		// Both widgets already send exactly these values, so this is not a
		// behavior change for them.
		referenceType = amaicall.ReferenceTypeNone
		referenceID = uuid.Nil
	}

	// create activeflow for the aicall
	af, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		customerID,
		uuid.Nil,
		fmactiveflow.ReferenceTypeAPI,
		uuid.Nil,
		uuid.Nil,
		nil,
		"",
		fmactiveflow.WebhookMethodNone,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create activeflow for aicall")
	}
	log.WithField("activeflow", af).Debugf("Created activeflow for aicall. activeflow_id: %s", af.ID)

	tmp, err := h.reqHandler.AIV1AIcallStart(
		ctx,
		aicallID,
		assistanceType,
		assistanceID,
		af.ID,
		referenceType,
		referenceID,
	)
	if err != nil {
		// best-effort cleanup of the orphaned activeflow
		if _, errDelete := h.reqHandler.FlowV1ActiveflowDelete(ctx, af.ID); errDelete != nil {
			log.Errorf("Could not delete orphaned activeflow. activeflow_id: %s, err: %v", af.ID, errDelete)
		}
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
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		// Listing has no target id to compare and nothing to overwrite, and it
		// is the only way a visitor could learn another visitor's resource id.
		// Refuse outright. Neither widget lists.
		//
		// GET /webchat_sessions already refuses direct; this asymmetry is what
		// left the aicall side enumerable.
		return nil, fmt.Errorf("%w: listing is not available for direct tokens", serviceerrors.ErrPermissionDenied)
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
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
		}
		if tmp.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
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
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
		}
		if c.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.AIV1AIcallDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIcallTerminate terminates the aicall.
func (h *serviceHandler) AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	c, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
		}
		if c.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not terminate the aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
