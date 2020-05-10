package callhandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"

	log "github.com/sirupsen/logrus"
)

// UpdateStatus updates the status and does actions.
func (h *callHandler) UpdateStatus(cn *channel.Channel) error {
	ctx := context.Background()

	status := call.GetStatusByChannelState(cn.State)
	if status != call.StatusRinging && status != call.StatusProgressing {
		// the call cares only riniging/progressing at here.
		// other statuses will be handled in the other func.
		return nil
	}

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err == dbhandler.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	// check status is updatable
	if call.IsUpdatableStatus(c.Status, status) == false {
		log.WithFields(log.Fields{
			"call_id":    c.ID,
			"old_status": c.Status,
			"new_status": status,
		}).Infof("The status is not updatable.")
		return nil
	}

	// we care only ringing/progress at here.
	if status == call.StatusRinging {
		return h.statusRinging(ctx, cn, c)
	} else if status == call.StatusProgressing {
		return h.statusProgressing(ctx, cn, c)
	}
	return nil
}

// statusRinging updates the call's status to ringing
func (h *callHandler) statusRinging(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	// update status
	if err := h.db.CallSetStatus(ctx, c.ID, call.StatusRinging, cn.TMRinging); err != nil {
		return err
	}
	return nil
}

// statusProgressing updates the call's status to progressing and does required actions.
func (h *callHandler) statusProgressing(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	// update status
	if err := h.db.CallSetStatus(ctx, c.ID, call.StatusProgressing, cn.TMAnswer); err != nil {
		return err
	}

	if c.Direction == call.DirectionIncoming {
		// nothing to do with incoming call at here.
		return nil
	}

	// todo: if the direciton is outgoing, we need to do some flow actions at here.

	return nil
}
