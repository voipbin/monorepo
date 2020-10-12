package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
)

// ConvertFMFlowToFlow returns converted data from fmflow.Flow to flow.Flow
func ConvertFMFlowToFlow(f *fmflow.Flow) *flow.Flow {

	return &flow.Flow{
		ID:     f.ID,
		UserID: f.UserID,

		Name:   f.Name,
		Detail: f.Detail,

		Actions: f.Actions,

		Persist: f.Persist,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}
}

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

	res := &flow.Flow{
		ID:      tmp.ID,
		UserID:  tmp.UserID,
		Name:    tmp.Name,
		Detail:  tmp.Detail,
		Actions: tmp.Actions,
		Persist: tmp.Persist,

		TMCreate: tmp.TMCreate,
		TMUpdate: tmp.TMUpdate,
		TMDelete: tmp.TMDelete,
	}

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

	res := ConvertFMFlowToFlow(flow)
	return res, nil
}

// FlowGetsByUserID gets the list of flow of the given user id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGetsByUserID(u *user.User, pageToken string, pageSize uint64) ([]*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
	})
	log.Debug("Getting a flow.")

	// get flows
	flows, err := h.reqHandler.FMFlowGets(u.ID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	var res []*flow.Flow
	for _, flow := range flows {
		tmp := ConvertFMFlowToFlow(&flow)
		res = append(res, tmp)
	}

	return res, nil
}
