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
		"func":       "Hangup",
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
	if errDestroy := h.bridgeHandler.Destroy(ctx, c.BridgeID); errDestroy != nil {
		// we don't care the error here. just write the log.
		log.Errorf("Could not destroy the bridge. err: %v", errDestroy)
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
	cc, err := h.UpdateHangupInfo(ctx, c.ID, reason, hangupBy)
	if err != nil {
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
	for _, callID := range cc.ChainedCallIDs {
		// hang up the call
		_, _ = h.HangingUp(ctx, callID, call.HangupReasonNormal)
		if err != nil {
			log.Errorf("Could not hangup the chained call. err: %v", err)
		}
	}

	// execute the master call execution
	h.handleMasterCallExecution(ctx, cc)

	return nil
}

// HangingUp starts hangup process.
// It sets the call status to the terminating and sends the hangup request to the Asterisk.
func (h *callHandler) HangingUp(ctx context.Context, id uuid.UUID, reason call.HangupReason) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "HangingUp",
		"call_id": id,
		"reason":  reason,
	})

	cause := call.ConvertHangupReasonToChannelCause(reason)
	log.Debugf("Hanging up the call. reason: %s, cause: %d", reason, cause)

	return h.hangingUpWithCause(ctx, id, cause)
}

// hangingUpWithCause starts hangup process.
// It sets the call status to the terminating and sends the hangup request to the Asterisk.
func (h *callHandler) hangingUpWithCause(ctx context.Context, id uuid.UUID, cause ari.ChannelCause) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "hangingUpWithCause",
		"call_id":       id,
		"channel_cause": cause,
	})
	log.Debugf("Hanging up the call. cause: %d", cause)

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}
	log.WithField("call", c).Debug("Call info.")

	if c.Status == call.StatusCanceling || c.Status == call.StatusHangup || c.Status == call.StatusTerminating {
		// already hanging up
		return c, nil
	}

	status := call.StatusTerminating
	if c.Direction == call.DirectionOutgoing && call.IsUpdatableStatus(c.Status, call.StatusCanceling) {
		// canceling
		// update call status to canceling
		status = call.StatusCanceling
	}

	// update call status
	log.Debugf("Updating call status for hanging up. status: %v", status)
	res, err := h.UpdateStatus(ctx, c.ID, status)
	if err != nil {
		// update status failed, just write log here. No need error handle here.
		log.Errorf("Could not update the call status for hangup. status: %v, err: %v", status, err)
		return nil, err
	}

	tmp, err := h.channelHandler.HangingUp(ctx, c.ChannelID, cause)
	if err != nil {
		log.Errorf("Could not hang up the call channel. err: %v", err)
		return nil, err
	}
	log.WithField("channel", tmp).Debugf("Hanging up the call channel. call_id: %s", c.ID)

	return res, nil
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

	if c.Data[call.DataTypeEarlyExecution] == "true" && c.Status == call.StatusRinging {
		log.Debug("The call's early execution is set and status is ringing. Consider the flow has started already.")
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
			log.Debugf("The hangup code is not retryable. code: %d", cn.HangupCause)
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

// handleMasterCallExecution handles master call execution.
// this is useful for connect calls.
// if the connecting(ougtoing) call failed the dialing, the master call will wait in the confbridge forever.
// so, to prevent that, we need to execute the master call's next action manually.
func (h *callHandler) handleMasterCallExecution(ctx context.Context, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "handleMasterCallExecution",
		"call_id": c.ID,
	})

	if c.Direction == call.DirectionIncoming {
		// nothing to do for the incoming call
		return
	}

	if c.MasterCallID == uuid.Nil {
		return
	}

	if c.Data[call.DataTypeEarlyExecution] == "true" {
		return
	}

	if c.Data[call.DataTypeExecuteNextMasterOnHangup] == "false" {
		return
	}

	if c.HangupReason == call.HangupReasonNormal {
		return
	}

	// execut the master call action next
	if errNext := h.reqHandler.CallV1CallActionNext(ctx, c.MasterCallID, false); errNext != nil {
		log.Errorf("Could not execute the master call's next action. err: %v", errNext)
	}
}
