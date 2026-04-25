package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"

	caoutplan "monorepo/bin-campaign-manager/models/outplan"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// outplanGet gets the outplan of the given id.
// It returns outplan if it succeed.
func (h *serviceHandler) outplanGet(ctx context.Context, id uuid.UUID) (*caoutplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "outplanGet",
		"outplan_id": id,
	})
	log.Debug("Getting an outplan.")

	// get outplan
	res, err := h.reqHandler.CampaignV1OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	return res, nil
}

// OutplanCreate is a service handler for outplan creation.
func (h *serviceHandler) OutplanCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	detail string,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Creating a new outplan.")

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.CampaignV1OutplanCreate(ctx, a.CustomerID, name, detail, source, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4)
	if err != nil {
		log.Errorf("Could not create a new outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanDelete deletes the outplan.
func (h *serviceHandler) OutplanDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"outplan_id":  id,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Deleting a outplan.")

	op, err := h.outplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info. err: %v", err)
		return nil, fmt.Errorf("could not get outplan info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, op.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanGetsByCustomerID gets the list of outplans of the given customer id.
// It returns list of outplans if it succeed.
func (h *serviceHandler) OutplanGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Getting a outplans.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get outplans
	filters := map[caoutplan.Field]any{
		caoutplan.FieldCustomerID: a.CustomerID,
	}
	outplans, err := h.reqHandler.CampaignV1OutplanList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get outplans info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplans info. err: %v", err)
	}

	// create result
	res := []*caoutplan.WebhookMessage{}
	for _, f := range outplans {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// OutplanGet gets the outplan of the given id.
// It returns outplan if it succeed.
func (h *serviceHandler) OutplanGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"outplan_id":  id,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Getting an outplan.")

	// get outplan
	tmp, err := h.outplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanUpdateBasicInfo updates the outplan's basic info.
// It returns updated outplan if it succeed.
func (h *serviceHandler) OutplanUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"outplan_id":  id,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Updating an outplan.")

	// get outplan
	op, err := h.outplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, op.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanUpdateDialInfo updates the outplan's dial info.
// It returns updated outplan if it succeed.
func (h *serviceHandler) OutplanUpdateDialInfo(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanUpdateDialInfo",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"outplan_id":  id,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log.Debug("Updating an outplan.")

	// get outplan
	op, err := h.outplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, op.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanUpdateDialInfo(ctx, id, source, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4)
	if err != nil {
		logrus.Errorf("Could not update the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
