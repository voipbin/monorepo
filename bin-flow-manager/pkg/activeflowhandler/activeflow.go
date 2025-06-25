package activeflowhandler

import (
	"context"
	"fmt"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceStop stops the service in the activeflow.
// the service should run in the current stack.
func (h *activeflowHandler) ServiceStop(ctx context.Context, id uuid.UUID, serviceID uuid.UUID) error {
	af, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", id)
	}

	if errPop := h.PopStackWithStackID(ctx, af, serviceID); errPop != nil {
		return errors.Wrapf(errPop, "could not pop the stack. stack_id: %s", serviceID)
	}

	return nil
}

// PushActions pushes the given actions in a new stack.
// pushed new stack will be executed in a next action request.
func (h *activeflowHandler) PushActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error) {

	af, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", id)
	}

	flowActions, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not generate the flow actions. activeflow_id: %s", id)
	}

	if errPush := h.PushStack(ctx, af, uuid.Nil, flowActions); errPush != nil {
		return nil, errors.Wrapf(errPush, "could not push the stack. activeflow_id: %s", id)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated activeflow info. activeflow_id: %s", id)
	}

	return res, nil
}

// AddActions adds the given actions in a current stack and current action.
func (h *activeflowHandler) AddActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error) {
	flowActions, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate the flow actions")
	}

	af, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", id)
	}

	if af.CurrentStackID == stack.IDMain {
		// Add actions in the main stack is not allowed.
		return nil, fmt.Errorf("current stack is main stack. activeflow_id: %s", id)
	}

	if errAdd := h.addActions(ctx, af, flowActions); errAdd != nil {
		return nil, errors.Wrapf(errAdd, "could not add the actions. activeflow_id: %s", id)
	}

	res, err := h.Get(ctx, af.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated activeflow info. activeflow_id: %s", af.ID)
	}

	return res, nil
}

// addActions adds the given actions in a current stack and current action.
func (h *activeflowHandler) addActions(ctx context.Context, af *activeflow.Activeflow, actions []action.Action) error {

	if errAdd := h.stackmapHandler.AddActions(af.StackMap, af.CurrentStackID, af.CurrentAction.ID, actions); errAdd != nil {
		return errors.Wrapf(errAdd, "could not add the actions. activeflow_id: %s", af.ID)
	}

	if errUpdate := h.updateStackProgress(ctx, af); errUpdate != nil {
		return errors.Wrapf(errUpdate, "could not update the activeflow. activeflow_id: %s", af.ID)
	}

	return nil
}

// PopStackWithStackID pop the given activeflow's current stack
func (h *activeflowHandler) PopStackWithStackID(ctx context.Context, af *activeflow.Activeflow, stackID uuid.UUID) error {
	if stackID != af.CurrentStackID {
		return fmt.Errorf("stack id is not matched. stack_id: %s, current_stack_id: %s", stackID, af.CurrentStackID)
	}

	tmp, err := h.stackmapHandler.PopStack(af.StackMap, af.CurrentStackID)
	if err != nil {
		return errors.Wrapf(err, "could not pop the stack. stack_id: %s", af.CurrentStackID)
	}

	// update forward actions
	af.ForwardStackID = tmp.ReturnStackID
	af.ForwardActionID = tmp.ReturnActionID

	// update activeflow
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after popped the stack. stack_id: %s", af.CurrentStackID)
	}

	return nil
}

// PopStack pop the given activeflow's current stack
func (h *activeflowHandler) PopStack(ctx context.Context, af *activeflow.Activeflow) error {

	if errPop := h.PopStackWithStackID(ctx, af, af.CurrentStackID); errPop != nil {
		return errors.Wrapf(errPop, "could not pop the stack. stack_id: %s", af.CurrentStackID)
	}

	return nil
}

// PushStack pushes the given action to the stack with a new stack
func (h *activeflowHandler) PushStack(ctx context.Context, af *activeflow.Activeflow, stackID uuid.UUID, actions []action.Action) error {
	if len(actions) == 0 {
		// no actions to push
		return nil
	}

	tmp, err := h.stackmapHandler.PushStackByActions(af.StackMap, stackID, actions, af.CurrentStackID, af.CurrentAction.ID)
	if err != nil {
		return errors.Wrapf(err, "could not push the actions. stack_id: %s", stackID)
	}

	// update forward actions
	af.ForwardStackID = tmp.ID
	af.ForwardActionID = tmp.Actions[0].ID

	// update activeflow
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after pushed the actions. stack_id: %s", stackID)
	}

	return nil
}
