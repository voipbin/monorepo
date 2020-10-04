package flowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/activeflow"
)

// FlowCreate creates a flow
// func (h *flowHandler) ActiveFlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error) {
func (h *flowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

	// get flow
	flow, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		logrus.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// create activeflow
	curTime := getCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID: callID,
		FlowID: flowID,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},

		Actions: flow.Actions,

		TMCreate: curTime,
		TMUpdate: curTime,
	}
	if err := h.db.ActiveFlowCreate(ctx, tmpAF); err != nil {
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	return af, nil
}
