package servicehandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
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
