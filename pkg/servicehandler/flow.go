package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
)

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(u *models.User, f *models.Flow) (*models.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":    u.ID,
		"name":    f.Name,
		"persist": f.Persist,
		"webhook": f.WebhookURI,
	})

	fmFlow := fmflow.CreateFlow(f)
	fmFlow.UserID = u.ID
	log.WithFields(
		logrus.Fields{
			"flow": fmFlow,
		},
	).Debugf("Creating a new flow. flow: %s", fmFlow.ID)

	tmp, err := h.reqHandler.FMFlowCreate(fmFlow)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertFlow()

	return res, nil
}

// FlowDelete deletes the flow of the given id.
func (h *serviceHandler) FlowDelete(u *models.User, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"flow_id":  id,
	})
	log.Debug("Deleting a flow.")

	// get flow
	flow, err := h.reqHandler.FMFlowGet(id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if u.HasPermission(models.UserPermissionAdmin) != true && flow.UserID != u.ID {
		log.Errorf("The user has no permission for this flow. user: %d, flow_user: %d", u.ID, flow.UserID)
		return fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.FMFlowDelete(id); err != nil {
		return err
	}

	return nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(u *models.User, id uuid.UUID) (*models.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"flow_id":  id,
	})
	log.Debug("Getting a flow.")

	// get flow
	flow, err := h.reqHandler.FMFlowGet(id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if u.HasPermission(models.UserPermissionAdmin) != true && flow.UserID != u.ID {
		log.Errorf("The user has no permission for this flow. user: %d, flow_user: %d", u.ID, flow.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	res := flow.ConvertFlow()
	return res, nil
}

// FlowGets gets the list of flow of the given user id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGets(u *models.User, size uint64, token string) ([]*models.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})
	log.Debug("Getting a flow.")

	if token == "" {
		token = getCurTime()
	}

	// get flows
	flows, err := h.reqHandler.FMFlowGets(u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	// create result
	res := []*models.Flow{}
	for _, flow := range flows {
		tmp := flow.ConvertFlow()
		res = append(res, tmp)
	}

	return res, nil
}

// FlowUpdate updates the flow info.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdate(u *models.User, f *models.Flow) (*models.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"flow":     f.ID,
	})
	log.Debug("Updating a flow.")

	// get flows
	tmpFlow, err := h.reqHandler.FMFlowGet(f.ID)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	// check the ownership
	if u.Permission != models.UserPermissionAdmin && u.ID != tmpFlow.UserID {
		log.Info("The user has no permission for this call.")
		return nil, fmt.Errorf("user has no permission")
	}

	reqFlow := fmflow.CreateFlow(f)
	res, err := h.reqHandler.FMFlowUpdate(reqFlow)
	if err != nil {
		logrus.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}

	resFlow := res.ConvertFlow()
	return resFlow, nil
}
