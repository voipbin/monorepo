package callhandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// updateStatusRinging updates the call's status to ringing
func (h *callHandler) updateStatusRinging(ctx context.Context, cn *channel.Channel, c *call.Call) error {

	// check status is updatable
	if call.IsUpdatableStatus(c.Status, call.StatusRinging) == false {
		log.WithFields(log.Fields{
			"call_id":    c.ID,
			"old_status": c.Status,
			"new_status": call.StatusRinging,
		}).Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusRinging)
	}

	// update status
	if err := h.db.CallSetStatus(ctx, c.ID, call.StatusRinging, cn.TMRinging); err != nil {
		return err
	}
	return nil
}

// updateStatusProgressing updates the call's status to progressing and does required actions.
func (h *callHandler) updateStatusProgressing(ctx context.Context, cn *channel.Channel, c *call.Call) error {

	// check status is updatable
	if call.IsUpdatableStatus(c.Status, call.StatusProgressing) == false {
		log.WithFields(log.Fields{
			"call_id":    c.ID,
			"old_status": c.Status,
			"new_status": call.StatusProgressing,
		}).Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusProgressing)
	}

	// update status
	if err := h.db.CallSetStatus(ctx, c.ID, call.StatusProgressing, cn.TMAnswer); err != nil {
		return err
	}

	if c.Direction == call.DirectionIncoming {
		// nothing to do with incoming call at here.
		return nil
	}

	// todo: if the direciton is outgoing, we need to do some flow actions at here.
	return h.ActionNext(c)
}
