package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
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

	// determine route-failover required
	if h.isRetryable(ctx, c, cn) {
		// retry the dial

		tmp, err := h.createFailoverChannel(ctx, c)
		if err == nil {
			log.Debugf("Created route failover channel succesfully. channel_id: %s", tmp.ChannelID)
			return nil
		}
		log.Errorf("Could not create a failover channel. Continue to hangup process. err: %v", err)
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
	if err := h.db.CallSetStatus(ctx, c.ID, status, h.util.GetCurTime()); err != nil {
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

// isRetryable returns true if the given call is dial retryable
func (h *callHandler) isRetryable(ctx context.Context, c *call.Call, cn *channel.Channel) bool {
	log := logrus.WithFields(logrus.Fields{
		"call_id":      c.ID,
		"channel_id":   cn.ID,
		"hangup_cause": cn.HangupCause,
	})

	// check the direction. the only outgoing calls are retryable
	if c.Direction != call.DirectionOutgoing {
		log.Debugf("The direction is not outgoing. Not retryable. direction: %s", c.Direction)
		return false
	}

	// check destination's type
	if c.Destination.Type != commonaddress.TypeTel {
		log.Debugf("The destination type is not tel type. type: %s", c.Destination.Type)
		return false
	}

	// check the channel cause code is retryable
	notRetryableCodes := []ari.ChannelCause{
		ari.ChannelCauseNormalClearing,
		ari.ChannelCauseUserBusy,
		ari.ChannelCauseNoAnswer,
		ari.ChannelCauseCallRejected,
		ari.ChannelCauseAnsweredElsewhere,

		ari.ChannelCauseCallDurationTimeout,
		ari.ChannelCauseCallAMD,
	}
	for _, code := range notRetryableCodes {
		if code == cn.HangupCause {
			log.Debugf("The")
			return false
		}
	}

	// check call's status
	if c.Status != call.StatusRinging && c.Status != call.StatusDialing {
		return false
	}

	// check call's dialroute
	_, err := h.getNextDialroute(ctx, c)
	if err != nil {
		// no more dialroute left
		log.Debugf("The call has no dialroute left to dial. call_id: %s", c.ID)
		return false
	}

	log.Debugf("The call is dial retryable. call_id: %s", c.ID)

	return true
}
