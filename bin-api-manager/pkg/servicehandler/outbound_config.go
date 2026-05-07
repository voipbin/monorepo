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

	if res.TMDelete != nil {
		return nil, fmt.Errorf("%w: deleted outbound config", serviceerrors.ErrStateInvalid)
	}

	return res, nil
}

// OutboundConfigCreate creates a new outbound config for the authenticated customer.
// Uses a.CustomerID from JWT — never a user-supplied customer_id (IDOR prevention).
func (h *serviceHandler) OutboundConfigCreate(ctx context.Context, a *auth.AuthIdentity, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigCreate",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
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

// OutboundConfigDelete deletes an outbound config, enforcing ownership.
func (h *serviceHandler) OutboundConfigDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigDelete",
		"customer_id": a.CustomerID,
		"id":          id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	cfg, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, cfg.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	deleted, err := h.reqHandler.CallV1OutboundConfigDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(deleted), nil
}

// OutboundConfigGet returns an outbound config by ID, enforcing ownership.
func (h *serviceHandler) OutboundConfigGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigGet",
		"customer_id": a.CustomerID,
		"id":          id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	cfg, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, cfg.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return cmoutboundconfig.ConvertWebhookMessage(cfg), nil
}

// OutboundConfigList returns outbound configs for the authenticated customer.
// Always uses a.CustomerID from JWT — never params.CustomerId (IDOR prevention).
func (h *serviceHandler) OutboundConfigList(ctx context.Context, a *auth.AuthIdentity, pageSize uint64, pageToken string) ([]*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigList",
		"customer_id": a.CustomerID,
		"page_size":   pageSize,
		"page_token":  pageToken,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if pageToken == "" {
		pageToken = h.utilHandler.TimeGetCurTime()
	}

	tmps, err := h.reqHandler.CallV1OutboundConfigList(ctx, a.CustomerID, pageSize, pageToken)
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

// OutboundConfigUpdate updates an outbound config, enforcing ownership.
func (h *serviceHandler) OutboundConfigUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutboundConfigUpdate",
		"customer_id": a.CustomerID,
		"id":          id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Fetch first to verify ownership before mutating.
	cfg, err := h.outboundConfigGet(ctx, id)
	if err != nil {
		log.Infof("Could not get outbound config. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, cfg.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	updated, err := h.reqHandler.CallV1OutboundConfigUpdate(ctx, id, req)
	if err != nil {
		log.Errorf("Could not update outbound config. err: %v", err)
		return nil, err
	}

	return cmoutboundconfig.ConvertWebhookMessage(updated), nil
}
