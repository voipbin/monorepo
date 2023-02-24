package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
)

// OutdialtargetCreate creates a new outdialtarget.
// It returns created outdialtarget if it succeed.
func (h *serviceHandler) OutdialtargetCreate(
	ctx context.Context,
	u *cscustomer.Customer,
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
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  outdialID,
		"data":        data,
	})
	log.Debug("Executing OutdialUpdateData.")

	// get outdial
	_, err := h.outdialGet(ctx, u, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialtargetGet(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  outdialID,
	})
	log.Debug("Executing OutdialtargetGet.")

	// get outdial
	_, err := h.outdialGet(ctx, u, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialtargetGetsByOutdialID(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetGetsByOutdialID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outdials.")

	if token == "" {
		token = h.utilHandler.GetCurTime()
	}

	// get outdial
	_, err := h.outdialGet(ctx, u, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialtargetDelete(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialtargetDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  outdialID,
	})
	log.Debug("Executing OutdialtargetDelete.")

	// get outdial
	od, err := h.reqHandler.OutdialV1OutdialGet(ctx, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != od.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
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
