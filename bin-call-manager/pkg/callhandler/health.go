package callhandler

import (
	"context"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// HealthCheck checks the given call is still vaild
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
		_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
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
	if c.Status == call.StatusHangup || c.TMDelete < dbhandler.DefaultTimeStamp || c.TMHangup < dbhandler.DefaultTimeStamp {
		// the call is done already. no need to check the health anymore.
		return
	}

	// get call's channel.
	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return
	}

	// validate channel info
	if cn.TMEnd < dbhandler.DefaultTimeStamp || cn.TMDelete < dbhandler.DefaultTimeStamp {
		// channel's status is not valid. consider it's being terminate.
		// increase retrycount and try again
		retryCount++
	} else {
		// the channel is valid and seems still on going.
		retryCount = 0
	}

	// send health check.
	if errHealth := h.reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, retryCount); errHealth != nil {
		log.Errorf("Could not send the call health check request. err: %v", errHealth)
		return
	}
}
