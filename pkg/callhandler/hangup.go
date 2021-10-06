package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// Hangup Hangup the call
func (h *callHandler) Hangup(cn *channel.Channel) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"channel_id": cn.ID,
	})

	// get call info
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		// nothing we can do
		log.Errorf("Could not get the call info from the db. err: %v", err)
		return nil
	}
	log = log.WithField("call_id", c.ID)

	// remove the call bridge
	if err := h.reqHandler.AstBridgeDelete(c.AsteriskID, c.BridgeID); err != nil {
		// we don't care the error here. just write the log.
		log.Errorf("Could not remove the bridge. err: %v", err)
	}

	// calculate hangup_reason, hangup_by
	reason := call.CalculateHangupReason(c.Status, cn.HangupCause)
	hangupBy := call.CalculateHangupBy(c.Status)

	// set hangup
	if err := h.HangupWithReason(ctx, c, reason, hangupBy, cn.TMEnd); err != nil {
		// we don't channel hangup here, because the channel has already gone.
		log.Errorf("Could not set the hangup reason. err: %v", err)
		return err
	}

	// hangup the chained call
	for _, callID := range c.ChainedCallIDs {
		chainedCall, err := h.db.CallGet(ctx, callID)
		if err != nil {
			log.WithField("chained_call_id", chainedCall).Errorf("Could not get chained call info. err: %v", err)
			continue
		}

		// hang up the call
		_ = h.HangingUp(chainedCall, ari.ChannelCauseNormalClearing)
	}

	return nil
}

// HangupWithReason set the hangup call with the given reason
func (h *callHandler) HangupWithReason(ctx context.Context, c *call.Call, reason call.HangupReason, hangupBy call.HangupBy, timestamp string) error {
	if err := h.db.CallSetHangup(ctx, c.ID, reason, hangupBy, timestamp); err != nil {
		// we don't channel hangup here, we are assumming the channel has already gone.
		return err
	}
	tmpCall, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		logrus.Errorf("Could not get hungup call data. call: %s, err: %v", c.ID, err)
		return nil
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeCallHungup, tmpCall.WebhookURI, tmpCall)

	promCallHangupTotal.WithLabelValues(string(c.Direction), string(c.Type), string(reason)).Inc()
	return nil
}

// HangingUp starts hangup process.
// It sets the call status to the terminating and sends the hangup request to the Asterisk.
func (h *callHandler) HangingUp(c *call.Call, cause ari.ChannelCause) error {
	ctx := context.Background()

	log := logrus.WithFields(logrus.Fields{
		"call":          c.ID,
		"asterisk":      c.AsteriskID,
		"channel":       c.ChannelID,
		"hangup reason": cause,
	})
	log.Debug("Hanging up the call.")

	if c.Status == call.StatusCanceling || c.Status == call.StatusHangup || c.Status == call.StatusTerminating {
		// already hanging up
		return nil
	}

	if c.Direction == call.DirectionOutgoing && call.IsUpdatableStatus(c.Status, call.StatusCanceling) {
		// canceling
		// update call status
		if err := h.db.CallSetStatus(ctx, c.ID, call.StatusCanceling, getCurTime()); err != nil {
			// update status failed, just write log here. No need error handle here.
			log.Errorf("Could not update the call status StatusCanceling for hangup. err: %v", err)
			return err
		}

	} else {
		// incoming and others
		// update call status
		if err := h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime()); err != nil {
			// update status failed, just write log here. No need error handle here.
			log.Errorf("Could not update the call status StatusTerminating for hangup. err: %v", err)
			return err
		}
	}

	// send hangup request
	if err := h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, cause); err != nil {
		// Send hangup request has failed. Something really wrong.
		log.Errorf("Could not send the hangup request for call hangup. err: %v", err)
		return err
	}

	return nil
}
