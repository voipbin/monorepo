package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ProviderCallCreate is a thin admin-gateway handler for an admin-triggered
// provider call. It:
//  1. Rejects direct-access tokens and callers without PermissionProjectSuperAdmin.
//  2. Validates the supplied provider_id exists (fail-fast before any call
//     record is created).
//  3. Forwards the full request to bin-route-manager which owns the
//     orchestration (temp-flow creation, metadata construction, Call
//     creation, and ProviderCall persistence).
//
// All orchestration moved to bin-route-manager because ProviderCall is a
// route-manager entity; keeping api-manager thin preserves the gateway
// responsibility (auth + validation + forward) without pulling call-manager
// and flow-manager coordination into the HTTP service layer.
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
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// permission — project admin only
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// fail-fast provider existence check — avoids asking route-manager to
	// create a Call record that would immediately fail at DialrouteList.
	if _, err := h.reqHandler.RouteV1ProviderGet(ctx, providerID); err != nil {
		log.Errorf("Could not find the provider. err: %v", err)
		return nil, fmt.Errorf("%w: provider not found", serviceerrors.ErrNotFound)
	}

	// forward to route-manager for orchestration
	tmp, err := h.reqHandler.RouteV1ProviderCallCreate(
		ctx,
		a.CustomerID,
		providerID,
		flowID,
		actions,
		source,
		destinations,
		anonymous,
	)
	if err != nil {
		log.Errorf("Could not create providercall. err: %v", err)
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
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.RouteV1ProviderCallDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete providercall. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
