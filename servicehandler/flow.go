package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(u *user.User, id uuid.UUID, name, detail string, actions []action.Action, persist bool) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":    u.ID,
		"flow":    id,
		"name":    name,
		"persist": persist,
	})
	log.Debug("Creating a new flow.")

	tmp, err := h.reqHandler.FMFlowCreate(u.ID, id, name, detail, actions, persist)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertFlow()

	return res, nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(u *user.User, id uuid.UUID) (*flow.Flow, error) {
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
	if u.HasPermission(user.PermissionAdmin) != true && flow.UserID != u.ID {
		log.Errorf("The user has no permission for this flow. user: %d, flow_user: %d", u.ID, flow.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	res := flow.ConvertFlow()
	return res, nil
}

// FlowGets gets the list of flow of the given user id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGets(u *user.User, size uint64, token string) ([]*flow.Flow, error) {
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
	res := []*flow.Flow{}
	for _, flow := range flows {
		tmp := flow.ConvertFlow()
		res = append(res, tmp)
	}

	return res, nil
}
