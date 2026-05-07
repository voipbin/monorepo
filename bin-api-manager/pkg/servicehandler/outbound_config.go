package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// outboundConfigGet fetches an outbound config by ID without permission checks.
func (h *serviceHandler) outboundConfigGet(ctx context.Context, id uuid.UUID) (*cmoutboundconfig.OutboundConfig, error) {
	res, err := h.reqHandler.CallV1OutboundConfigGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get outbound config: %w", err)
	}

	if res == nil || res.ID == uuid.Nil {
		return nil, fmt.Errorf("not found: outbound_config %s", id)
	}

	if res.TMDelete != nil {
		return nil, fmt.Errorf("%w: deleted outbound config", serviceerrors.ErrStateInvalid)
	}

	return res, nil
}

// OutboundConfigCreate creates a new outbound config for a given customer.
// Admin-only (requires PermissionProjectSuperAdmin).
func (h *serviceHandler) OutboundConfigCreate(ctx context.Context, a *auth.AuthIdentity, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigCreate",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.CallV1OutboundConfigCreate(ctx, a.CustomerID, req)
	if err != nil {
		log.Errorf("Could not create outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(res), nil
}

// OutboundConfigDelete deletes an outbound config by ID.
// Admin-only (requires PermissionProjectSuperAdmin).
func (h *serviceHandler) OutboundConfigDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigDelete",
		"customer_id": a.CustomerID,
		"id":          id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	deleted, err := h.reqHandler.CallV1OutboundConfigDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(deleted), nil
}

// OutboundConfigGet returns an outbound config by ID.
// Admin-only (requires PermissionProjectSuperAdmin).
func (h *serviceHandler) OutboundConfigGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigGet",
		"customer_id": a.CustomerID,
		"id":          id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	cfg, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(cfg), nil
}

// OutboundConfigList returns all outbound configs across all customers.
// Admin-only (requires PermissionProjectSuperAdmin).
func (h *serviceHandler) OutboundConfigList(ctx context.Context, a *auth.AuthIdentity, pageSize uint64, pageToken string) ([]*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "OutboundConfigList",
		"page_size":  pageSize,
		"page_token": pageToken,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if pageToken == "" {
		pageToken = h.utilHandler.TimeGetCurTime()
	}

	// Admin list: pass uuid.Nil to list across all customers.
	tmps, err := h.reqHandler.CallV1OutboundConfigList(ctx, uuid.Nil, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list outbound configs. err: %v", err)
		return nil, err
	}

	res := make([]*cmoutboundconfig.WebhookMessage, 0, len(tmps))
	for i := range tmps {
		res = append(res, cmoutboundconfig.ConvertWebhookMessage(&tmps[i]))
	}

	return res, nil
}

// OutboundConfigSelfGet returns the outbound config for the authenticated customer.
// Resolves the config from the JWT customer ID — no explicit ID required.
func (h *serviceHandler) OutboundConfigSelfGet(ctx context.Context, a *auth.AuthIdentity) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigSelfGet",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmps, err := h.reqHandler.CallV1OutboundConfigList(ctx, a.CustomerID, 1, h.utilHandler.TimeGetCurTime())
	if err != nil {
		log.Errorf("Could not list outbound configs. err: %v", err)
		return nil, err
	}

	if len(tmps) == 0 {
		return nil, fmt.Errorf("%w: outbound config not found for customer %s", serviceerrors.ErrNotFound, a.CustomerID)
	}

	return cmoutboundconfig.ConvertWebhookMessage(&tmps[0]), nil
}

// OutboundConfigSelfUpdate updates the outbound config for the authenticated customer.
// Resolves the config from the JWT customer ID — no explicit ID required.
func (h *serviceHandler) OutboundConfigSelfUpdate(ctx context.Context, a *auth.AuthIdentity, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigSelfUpdate",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Resolve the config ID by fetching the customer's own config.
	cfg, err := h.OutboundConfigSelfGet(ctx, a)
	if err != nil {
		log.Infof("Could not get own outbound config. err: %v", err)
		return nil, err
	}

	updated, err := h.reqHandler.CallV1OutboundConfigUpdate(ctx, cfg.ID, req)
	if err != nil {
		log.Errorf("Could not update outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(updated), nil
}

// OutboundConfigUpdate updates an outbound config by ID.
// Admin-only (requires PermissionProjectSuperAdmin).
func (h *serviceHandler) OutboundConfigUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "OutboundConfigUpdate",
		"id":   id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Fetch first to verify the config exists before mutating.
	_, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	updated, err := h.reqHandler.CallV1OutboundConfigUpdate(ctx, id, req)
	if err != nil {
		log.Errorf("Could not update outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(updated), nil
}
