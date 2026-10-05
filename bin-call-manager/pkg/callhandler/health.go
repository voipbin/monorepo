package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// HealthCheck checks the given call is still valid
// and hangup the call if the call is not valid over the default retry count.
func (h *callHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "HealthCheck",
		"call_id":     id,
		"retry_count": retryCount,
	})

	// if the retry count is bigger than defaultHealthMaxRetryCount,
	// hangup the call
	if retryCount > defaultHealthMaxRetryCount {
		log.Infof("Exceeded max call health check retry count. Hanging up the call. call_id: %s", id)
		h.healthHangup(ctx, id, retryCount)
		return
	}

	// get call info.
	c, err := h.Get(ctx, id)
	if err != nil {
		// failed to get call info. consider database failure.
		// in this case, nothing we can do. just write error and no need retry
		log.Errorf("Could not call info. err: %v", err)
		return
	}

	// validate call info
	if c.Status == call.StatusHangup || c.TMDelete != nil || c.TMHangup != nil {
		// the call is done already. no need to check the health anymore.
		return
	}

	// get call's channel.
	// we read the database handler directly, not the channel handler: the channel handler waits for a missing channel
	// and reports every failure as not-found, but here we have to tell a missing channel row from a read failure.
	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	switch {
	case errors.Is(err, dbhandler.ErrNotFound):
		// the channel row does not exist (yet).
		// count it like an ended channel and keep watching. a row that appears later resets the count.
		retryCount++

	case err != nil:
		// a read failure says nothing about the channel. no need to retry.
		log.Errorf("Could not get channel info. err: %v", err)
		return

	case cn.TMEnd != nil || cn.TMDelete != nil:
		// channel's status is not valid. consider it's being terminate.
		// increase retrycount and try again
		retryCount++

	default:
		// the channel is valid and seems still on going.
		retryCount = 0
	}

	// send health check.
	if errHealth := h.reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, retryCount); errHealth != nil {
		log.Errorf("Could not send the call health check request. err: %v", errHealth)
		return
	}
}

// healthHangup finishes a call that stayed bound to an ended or a missing channel for more than the max retry count.
//
// The health check counts a retry only while the channel is ended or missing. HangingUp is the right action for
// most calls, but it returns early for a call that is already canceling or terminating, and a call whose channel
// row never existed has nothing to hang up. This function handles those two cases and leaves everything else to
// HangingUp.
func (h *callHandler) healthHangup(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "healthHangup",
		"call_id":     id,
		"retry_count": retryCount,
	})

	// read the latest state from the database, not the cache, right before acting.
	c, err := h.db.CallGetFromDB(ctx, id)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return
	}

	// the call has finished meanwhile. nothing to do.
	if c.Status == call.StatusHangup || c.TMHangup != nil || c.TMDelete != nil {
		return
	}

	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	switch {
	case errors.Is(err, dbhandler.ErrNotFound):
		// the channel row is missing after the max retry count. only a dialing call (or a call that a hangup
		// request moved to canceling) can be without the channel row: the dial has never started.
		if c.Status == call.StatusDialing || c.Status == call.StatusCanceling {
			h.hangupFailedCall(ctx, c)
			return
		}

		// any other status always has the channel row. nothing to decide here.
		log.Errorf("The channel row is missing for the call in an unexpected status. status: %s, channel_id: %s", c.Status, c.ChannelID)
		return

	case err != nil:
		// a read failure says nothing about the channel. no need to retry.
		log.Errorf("Could not get channel info. channel_id: %s, err: %v", c.ChannelID, err)
		return

	case cn.TMEnd == nil && cn.TMDelete == nil:
		// the channel is alive: the failures we counted were transient. start over.
		log.Infof("The channel is alive. Restarting the health check. channel_id: %s", cn.ID)
		if errHealth := h.reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, 0); errHealth != nil {
			log.Errorf("Could not send the call health check request. err: %v", errHealth)
		}
		return

	case c.Status == call.StatusCanceling || c.Status == call.StatusTerminating:
		// HangingUp returns early for these statuses. finish the call through Hangup,
		// the function that the ChannelDestroyed event would have run.
		if _, errHangup := h.Hangup(ctx, cn); errHangup != nil {
			log.Errorf("Could not hangup the call. err: %v", errHangup)
		}
		return
	}

	// the other statuses keep the existing behavior.
	_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
}

// hangupFailedCall finalizes a call whose channel never appeared.
//
// The groupcall is not notified here: a groupcall caller that could not create the call already updates its own
// counters on the create error.
func (h *callHandler) hangupFailedCall(ctx context.Context, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "hangupFailedCall",
		"call_id": c.ID,
	})
	log.Infof("The call has no channel. Hanging up the call. status: %s, channel_id: %s", c.Status, c.ChannelID)

	if _, _, err := h.UpdateHangupInfo(ctx, c.ID, "", call.HangupReasonFailed, call.HangupByLocal); err != nil {
		log.Errorf("Could not update the hangup info. err: %v", err)
		return
	}

	// a call created with a dummy activeflow has uuid.Nil: nothing to stop.
	// (Hangup stops the activeflow unconditionally. the guard follows the earlier failure path of
	// CreateCallOutgoing and avoids a request that cannot succeed.)
	if c.ActiveflowID != uuid.Nil {
		if _, err := h.reqHandler.FlowV1ActiveflowStop(ctx, c.ActiveflowID); err != nil {
			log.Errorf("Could not stop the activeflow. activeflow_id: %s, err: %v", c.ActiveflowID, err)
		}
	}
}
