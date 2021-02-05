package flowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
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
func (h *flowHandler) FlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error) {

	if flow.ID == uuid.Nil {
		flow.ID = uuid.Must(uuid.NewV4())
	}

	// set action id
	for i := range flow.Actions {
		flow.Actions[i].ID = uuid.Must(uuid.NewV4())
	}

	switch {
	case persist == true:
		if err := h.db.FlowCreate(ctx, flow); err != nil {
			return nil, err
		}

	default:
		if err := h.db.FlowSetToCache(ctx, flow); err != nil {
			return nil, err
		}
	}

	resFlow, err := h.FlowGet(ctx, flow.ID)
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
func (h *flowHandler) FlowDelete(ctx context.Context, id uuid.UUID) error {
	err := h.db.FlowDelete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}
