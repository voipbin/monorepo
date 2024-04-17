package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// updateStatusRinging updates the call's status to ringing
func (h *callHandler) updateStatusRinging(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "updateStatusRinging",
		"call_id":    c.ID,
		"old_status": c.Status,
		"new_status": call.StatusRinging,
	})

	// check status is updatable
	if !call.IsUpdatableStatus(c.Status, call.StatusRinging) {
		log.Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusRinging)
	}

	cc, err := h.UpdateStatus(ctx, c.ID, call.StatusRinging)
	if err != nil {
		log.Errorf("Could not update the call status. err: %v", err)
		return err
	}

	if cc.Data[call.DataTypeEarlyExecution] == "true" {
		log.Debugf("The call has early execution. Executing the flow. early_execution: %s", cc.Data[call.DataTypeEarlyExecution])
		return h.ActionNext(ctx, cc)
	}

	return nil
}

// updateStatusProgressing updates the call's status to progressing and does required actions.
func (h *callHandler) updateStatusProgressing(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "updateStatusProgressing",
		"call_id":    c.ID,
		"old_status": c.Status,
		"new_status": call.StatusProgressing,
	})

	// check status is updatable
	if !call.IsUpdatableStatus(c.Status, call.StatusProgressing) {
		log.Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusProgressing)
	}

	res, err := h.UpdateStatus(ctx, c.ID, call.StatusProgressing)
	if err != nil {
		log.Errorf("Could not update the call status. err: %v", err)
		return err
	}

	if res.Direction == call.DirectionIncoming {
		// nothing to do with incoming call at here.
		return nil
	}

	// check the groupcall info and answer the groupcall
	if res.GroupcallID != uuid.Nil {
		log.Debugf("The call has groupcall id. Answering the groupcall. groupcall_id: %s", res.GroupcallID)
		if errAnswer := h.groupcallHandler.AnswerCall(ctx, res.GroupcallID, res.ID); errAnswer != nil {
			log.Errorf("Could not update the group dial answer call id. err: %v", errAnswer)
		}
	}

	// check the confbridge info and answer the confbridge
	if res.ConfbridgeID != uuid.Nil {
		log.Debugf("The call has confbridge id. Answering the confbridge. groupcall_id: %s", res.GroupcallID)
		if errAnswer := h.confbridgeHandler.Answer(ctx, res.ConfbridgeID); errAnswer != nil {
			log.Errorf("Could not answer the confbridge. err: %v", errAnswer)
		}
	}

	if c.Data[call.DataTypeEarlyExecution] == "true" {
		log.Debugf("The call has early execution. Consider the flow execution is already on going. early_execution: %s", c.Data[call.DataTypeEarlyExecution])
		return nil
	}

	return h.ActionNext(ctx, res)
}
