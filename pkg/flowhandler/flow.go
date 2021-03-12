package flowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// FlowGet returns flow
func (h *flowHandler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	resFlow, err := h.db.FlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowCreate creates a flow
func (h *flowHandler) FlowCreate(ctx context.Context, f *flow.Flow) (*flow.Flow, error) {

	f.ID = uuid.Must(uuid.NewV4())

	// set action id
	for i := range f.Actions {
		f.Actions[i].ID = uuid.Must(uuid.NewV4())
	}

	switch {
	case f.Persist == true:
		if err := h.db.FlowCreate(ctx, f); err != nil {
			return nil, err
		}

	default:
		if err := h.db.FlowSetToCache(ctx, f); err != nil {
			return nil, err
		}
	}

	resFlow, err := h.FlowGet(ctx, f.ID)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowGetsByUserID returns list of flows
func (h *flowHandler) FlowGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*flow.Flow, error) {

	flows, err := h.db.FlowGetsByUserID(ctx, userID, token, limit)
	if err != nil {
		logrus.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return flows, nil
}

// FlowUpdate updates the flow info and return the updated flow
func (h *flowHandler) FlowUpdate(ctx context.Context, f *flow.Flow) (*flow.Flow, error) {
	logrus.WithFields(
		logrus.Fields{
			"flow": f,
		},
	).Debugf("Updating flow. flow: %s", f.ID)

	// set action id
	for i := range f.Actions {
		f.Actions[i].ID = uuid.Must(uuid.NewV4())
	}

	if err := h.db.FlowUpdate(ctx, f); err != nil {
		logrus.Errorf("Could not update the flow info. err: %v", err)
		return nil, err
	}

	res, err := h.db.FlowGet(ctx, f.ID)
	if err != nil {
		logrus.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FlowDelete delets the given flow
// And it also removes the related flow_id from the number-manager
func (h *flowHandler) FlowDelete(ctx context.Context, id uuid.UUID) error {
	err := h.db.FlowDelete(ctx, id)
	if err != nil {
		return err
	}

	// send related flow-id clean up request to the number-manager
	if err := h.reqHandler.NMNumberFlowDelete(id); err != nil {
		logrus.Errorf("Could not clean up the flow_id from the number-manager. err: %v", err)
		// we don't return the err here, because the flow has been removed already.
		// and the numbers which have the removed flow-id are OK too, because
		// when the call-manager request the flow, that request will be failed too.
	}

	return nil
}
