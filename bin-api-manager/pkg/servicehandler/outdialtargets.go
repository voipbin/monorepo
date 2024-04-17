package servicehandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// OutdialtargetCreate creates a new outdialtarget.
// It returns created outdialtarget if it succeed.
func (h *serviceHandler) OutdialtargetCreate(
	ctx context.Context,
	a *amagent.Agent,
	outdialID uuid.UUID,
	name string,
	detail string,
	data string,
	destination0 *commonaddress.Address,
	destination1 *commonaddress.Address,
	destination2 *commonaddress.Address,
	destination3 *commonaddress.Address,
	destination4 *commonaddress.Address,
) (*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  outdialID,
		"data":        data,
	})
	log.Debug("Executing OutdialUpdateData.")

	// get outdial
	od, err := h.outdialGet(ctx, a, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// create
	tmp, err := h.reqHandler.OutdialV1OutdialtargetCreate(
		ctx,
		outdialID,
		name,
		detail,
		data,
		destination0,
		destination1,
		destination2,
		destination3,
		destination4,
	)
	if err != nil {
		logrus.Errorf("Could not create the outdialtarget. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialtargetGet gets an outdialtarget.
// It returns outdialtarget if it succeed.
func (h *serviceHandler) OutdialtargetGet(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  outdialID,
	})
	log.Debug("Executing OutdialtargetGet.")

	// get outdial
	od, err := h.outdialGet(ctx, a, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get outdialtarget
	tmp, err := h.reqHandler.OutdialV1OutdialtargetGet(ctx, outdialtargetID)
	if err != nil {
		log.Errorf("Could not get outdialtarget info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdialtarget info. err: %v", err)
	}

	// check the outdial_id
	if tmp.OutdialID != outdialID {
		log.Errorf("The outdial_id is wrong. outdial_id: %s", tmp.OutdialID)
		return nil, fmt.Errorf("wrong outdial_id. outdial_id: %s", tmp.OutdialID)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialtargetGetsByOutdialID gets the list of outdialtargets of the given outdial id.
// It returns list of outdialtargets if it succeed.
func (h *serviceHandler) OutdialtargetGetsByOutdialID(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetGetsByOutdialID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outdials.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get outdial
	od, err := h.outdialGet(ctx, a, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get targets
	targets, err := h.reqHandler.OutdialV1OutdialtargetGetsByOutdialID(ctx, outdialID, token, size)
	if err != nil {
		log.Errorf("Could not get outdials info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdials info. err: %v", err)
	}

	// create result
	res := []*omoutdialtarget.WebhookMessage{}
	for _, f := range targets {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// OutdialtargetDelete deletes an outdialtarget.
// It returns deleted outdialtarget if it succeed.
func (h *serviceHandler) OutdialtargetDelete(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  outdialID,
	})
	log.Debug("Executing OutdialtargetDelete.")

	// get outdial
	od, err := h.outdialGet(ctx, a, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get outdialtarget
	ot, err := h.reqHandler.OutdialV1OutdialtargetGet(ctx, outdialtargetID)
	if err != nil {
		log.Errorf("Could not get outdialtarget info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdialtarget info. err: %v", err)
	}

	// check the outdial_id
	if ot.OutdialID != outdialID {
		log.Errorf("The outdial_id is wrong. outdial_id: %s", ot.OutdialID)
		return nil, fmt.Errorf("wrong outdial_id. outdial_id: %s", ot.OutdialID)
	}

	// delete
	tmp, err := h.reqHandler.OutdialV1OutdialtargetDelete(ctx, outdialtargetID)
	if err != nil {
		logrus.Errorf("Could not delete the outdialtarget. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
