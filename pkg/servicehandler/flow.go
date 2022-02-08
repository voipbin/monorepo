package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(u *cscustomer.Customer, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"name":        name,
		"persist":     persist,
	})

	log.WithField("actions", actions).Debug("Creating a new flow.")
	tmp, err := h.reqHandler.FMV1FlowCreate(ctx, u.ID, fmflow.TypeFlow, name, detail, actions, persist)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowDelete deletes the flow of the given id.
func (h *serviceHandler) FlowDelete(u *cscustomer.Customer, id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"flow_id":     id,
	})
	log.Debug("Deleting a flow.")

	// get flow
	f, err := h.reqHandler.FMV1FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != f.CustomerID {
		log.Errorf("The customer has no permission for this flow. customer: %s, flow_customer: %s", u.ID, f.CustomerID)
		return fmt.Errorf("customer has no permission")
	}

	if err := h.reqHandler.FMV1FlowDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"flow_id":     id,
	})
	log.Debug("Getting a flow.")

	// get flow
	tmp, err := h.reqHandler.FMV1FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Errorf("The customer has no permission for this flow. customer: %s, flow_customer: %s", u.ID, tmp.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGets gets the list of flow of the given customer id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGets(u *cscustomer.Customer, size uint64, token string) ([]*fmflow.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a flow.")

	if token == "" {
		token = getCurTime()
	}

	// get flows
	flows, err := h.reqHandler.FMV1FlowGets(ctx, u.ID, fmflow.TypeFlow, token, size)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
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
func (h *serviceHandler) FlowUpdate(u *cscustomer.Customer, f *fmflow.Flow) (*fmflow.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"flow":        f.ID,
	})
	log.Debug("Updating a flow.")

	// get flows
	tmpFlow, err := h.reqHandler.FMV1FlowGet(ctx, f.ID)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpFlow.CustomerID {
		log.Info("The customer has no permission for this call.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.FMV1FlowUpdate(ctx, f)
	if err != nil {
		logrus.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
