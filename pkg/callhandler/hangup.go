package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
)

// Hangup Hangup the call
func (h *callHandler) Hangup(cn *channel.Channel) error {
	ctx := context.Background()
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		// nothing we can do
		return nil
	}

	// calculate hangup_reason, hangup_by
	reason := call.CalculateHangupReason(c.Status, cn.HangupCause)
	hangupBy := call.CalculateHangupBy(c.Status)

	// update call
	if err := h.db.CallSetHangup(ctx, c.ID, reason, hangupBy, cn.TMEnd); err != nil {
		// we don't channel hangup here, because the channel has already gone.
		return err
	}
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
