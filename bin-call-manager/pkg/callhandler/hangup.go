package callhandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// Hangup Hangup the call
func (h *callHandler) Hangup(ctx context.Context, cn *channel.Channel) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Hangup",
		"channel": cn,
	})

	// get call info
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		// nothing we can do
		log.Errorf("Could not get the call info from the db. err: %v", err)
		return nil, errors.Wrap(err, "could not get call info from the db")
	}
	log = log.WithField("call", c)
	log.Debugf("Hanging up the call. call_id: %s, channel_id: %s", c.ID, cn.ID)

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
			return tmp, nil
		}
		log.Errorf("Could not create a failover channel. Continue to hangup process. err: %v", err)
	}

	// calculate hangup_reason, hangup_by
	reason := call.CalculateHangupReason(c.Direction, c.Status, cn.HangupCause)
	hangupBy := call.CalculateHangupBy(c.Status)

	// set hangup
	res, err := h.UpdateHangupInfo(ctx, c.ID, reason, hangupBy)
	if err != nil {
		// we don't channel hangup here, because the channel has already gone.
		log.Errorf("Could not set the hangup reason. err: %v", err)
		return nil, err
	}

	// check the call is part of groupcall
	if res.GroupcallID != uuid.Nil {
		log.Debugf("The call has groupcall id. Updating groupcall hangup call info. groupcall_id: %s", res.GroupcallID)
		go func(id uuid.UUID) {
			if errReq := h.reqHandler.CallV1GroupcallHangupCall(ctx, id); errReq != nil {
				// we don't do any error handle here.
				// just write the log.
				log.Errorf("Could not hangup the groupcall. err: %v", err)
			}
		}(res.GroupcallID)
	}

	// send activeflow stop
	log.Debugf("Stopping the activeflow. activeflow_id: %s", c.ActiveflowID)
	_, err = h.reqHandler.FlowV1ActiveflowStop(ctx, c.ActiveflowID)
	if err != nil {
		// we don't do anything here. just write the log
		log.Errorf("Could not stop the activeflow correctly. err: %v", err)
	}

	// hangup the chained call
	for _, callID := range res.ChainedCallIDs {
		// hang up the call
		log.Debugf("Haningup the chained call. chained_call_id: %s", callID)
		tmp, err := h.HangingUp(ctx, callID, call.HangupReasonNormal)
		if err != nil {
			// we don't do any error handle here.
			// just write the log
			log.Errorf("Could not hangup the chained call. err: %v", err)
			continue
		}
		log.WithField("chained_call", tmp).Debugf("Hanging up the chained call. chained_call_id: %s", tmp.ID)
	}

	return res, nil
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

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}
	log.WithField("call", c).Debugf("Hanging up the call. call_id: %s, cause: %d", c.ID, cause)

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
	res, err := h.UpdateStatus(ctx, c.ID, status)
	if err != nil {
		// update status failed, just write log here. No need error handle here.
		log.Errorf("Could not update the call status for hangup. status: %v, err: %v", status, err)
		return nil, err
	}

	cn, err := h.channelHandler.HangingUp(ctx, c.ChannelID, cause)
	if err != nil {
		log.Errorf("Could not hang up the call channel. err: %v", err)
		return nil, err
	}
	log.WithField("channel", cn).Debugf("Hanging up the call channel. call_id: %s, channel_id: %s", c.ID, cn.ID)

	if cn.TMEnd != nil {
		// the channel is already ended. just hang up the call.
		log.Debugf("The channel is already hungup. Hangup the call immediately. channel_id: %s, channel_state: %s", cn.ID, cn.State)
		tmp, err := h.Hangup(ctx, cn)
		if err != nil {
			// we could not hangup the call.
			// but just return the nil error here, because we succeeded hanging the call already.
			log.Errorf("Could not hangup the call correctly. err: %v", err)
			return res, nil
		}
		log.WithField("call", tmp).Debugf("Hungup the call. call_id: %s", res.ID)

		// update the res to the new call struct
		res = tmp
	}

	return res, nil
}

// isRetryable returns true if the given call is dial retryable
func (h *callHandler) isRetryable(ctx context.Context, c *call.Call, cn *channel.Channel) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":         "isRetryable",
		"call":         c,
		"channel":      cn,
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

// hangingupWithReference hanging up the call with the same reason of the given reference id.
func (h *callHandler) hangingupWithReference(ctx context.Context, c *call.Call, referenceID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "hangingupWithReference",
		"call_id":      c.ID,
		"reference_id": referenceID,
	})

	// get call info
	referenceCall, err := h.Get(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get referenced call info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get reference call info.")
	}
	log.WithField("reference_call", referenceCall).Debugf("Found referenced call info. reference_id: %s", referenceCall.ID)

	if referenceCall.Status != call.StatusHangup {
		log.Infof("The reference call is not hung up yet. Hang up the call with normal reason. reference_id: %s", referenceCall.ID)
		return nil, fmt.Errorf("the reference call is not hung up")
	}

	// get channel
	ch, err := h.channelHandler.Get(ctx, referenceCall.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get reference call channel.")
	}

	// hangup the call
	res, err := h.hangingUpWithCause(ctx, c.ID, ch.HangupCause)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		return nil, errors.Wrap(err, "Could not hangup the call.")
	}
	log.WithField("call", res).Debugf("Hanging up the call. call_id: %s", res.ID)

	return res, nil
}
