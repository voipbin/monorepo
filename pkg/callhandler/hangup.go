package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Hangup Hangup the call
func (h *callHandler) Hangup(ctx context.Context, cn *channel.Channel) error {
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
	if err := h.reqHandler.AstBridgeDelete(ctx, c.AsteriskID, c.BridgeID); err != nil {
		// we don't care the error here. just write the log.
		log.Errorf("Could not remove the bridge. err: %v", err)
	}

	// calculate hangup_reason, hangup_by
	reason := call.CalculateHangupReason(c.Direction, c.Status, cn.HangupCause)
	hangupBy := call.CalculateHangupBy(c.Status)

	// set hangup
	if err := h.HangupWithReason(ctx, c, reason, hangupBy, cn.TMEnd); err != nil {
		// we don't channel hangup here, because the channel has already gone.
		log.Errorf("Could not set the hangup reason. err: %v", err)
		return err
	}

	// send activeflow delete
	log.Debugf("Deleting activeflow. activeflow_id: %s", c.ActiveFlowID)
	_, err = h.reqHandler.FlowV1ActiveflowDelete(ctx, c.ActiveFlowID)
	if err != nil {
		// we don't do anything here. just write log only
		log.Errorf("Could not delete activeflow correctly. err: %v", err)
	}

	// hangup the chained call
	for _, callID := range c.ChainedCallIDs {
		// hang up the call
		if err := h.HangingUp(ctx, callID, ari.ChannelCauseNormalClearing); err != nil {
			log.Errorf("Could not hangup the chained call. err: %v", err)
		}
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
	h.notifyHandler.PublishWebhookEvent(ctx, tmpCall.CustomerID, call.EventTypeCallHungup, tmpCall)

	promCallHangupTotal.WithLabelValues(string(c.Direction), string(c.Type), string(reason)).Inc()
	return nil
}

// HangingUp starts hangup process.
// It sets the call status to the terminating and sends the hangup request to the Asterisk.
func (h *callHandler) HangingUp(ctx context.Context, id uuid.UUID, cause ari.ChannelCause) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "HangingUp",
		"call_id":       id,
		"hangup reason": cause,
	})
	log.Debug("Hanging up the call.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}
	log.WithField("call", c).Debug("Call info.")

	if c.Status == call.StatusCanceling || c.Status == call.StatusHangup || c.Status == call.StatusTerminating {
		// already hanging up
		return nil
	}

	status := call.StatusTerminating
	if c.Direction == call.DirectionOutgoing && call.IsUpdatableStatus(c.Status, call.StatusCanceling) {
		// canceling
		// update call status to canceling
		status = call.StatusCanceling
	}

	// update call status
	log.Debugf("Updating call status for hangup. status: %v", status)
	if err := h.db.CallSetStatus(ctx, c.ID, status, dbhandler.GetCurTime()); err != nil {
		// update status failed, just write log here. No need error handle here.
		log.Errorf("Could not update the call status for hangup. status: %v, err: %v", status, err)
		return err
	}

	// send hangup request
	if err := h.reqHandler.AstChannelHangup(ctx, c.AsteriskID, c.ChannelID, cause, 0); err != nil {
		// Send hangup request has failed. Something really wrong.
		log.Errorf("Could not send the hangup request for call hangup. err: %v", err)
		return err
	}

	return nil
}
