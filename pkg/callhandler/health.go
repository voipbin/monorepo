package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// HealthCheck checks the given call is still vaild
// and hangup the call if the call is not valid over the default retry count.
func (h *callHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "HealthCheck",
		"call_id":     id,
		"retry_count": retryCount,
	})

	// validate the call.
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not call info. err: %v", err)
		return
	}
	if c.Status == call.StatusHangup || c.TMDelete < dbhandler.DefaultTimeStamp || c.TMHangup < dbhandler.DefaultTimeStamp {
		// the call is done already. no need to check the health anymore.
		return
	}

	// validate call's channel.
	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return
	}
	if cn.TMEnd < dbhandler.DefaultTimeStamp || cn.TMDelete < dbhandler.DefaultTimeStamp {
		retryCount++
	} else {
		retryCount = 0
	}

	// if the retry count is bigger than defaultHealthMaxRetryCount,
	// hangup the call
	if retryCount > defaultHealthMaxRetryCount {
		log.WithField("call", c).Infof("Exceeded max call health check retry count. Hanging up the call. call_id: %s", c.ID)
		_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		return
	}

	// send health check.
	if errHealth := h.reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, retryCount); errHealth != nil {
		log.Errorf("Could not send the call health check request. err: %v", errHealth)
		return
	}
}
