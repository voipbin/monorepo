package groupcallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
)

// HangingupOthers hangs up the call except answered call.
func (h *groupcallHandler) HangingupOthers(ctx context.Context, gc *groupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "HangingupOthers",
		"groupcall": gc,
	})

	// check the chained groupcalls
	for _, groupcallID := range gc.GroupcallIDs {
		go func(id uuid.UUID) {
			log.Debugf("Sending groupcall hangup others. groupcall_id: %s", id)
			_ = h.reqHandler.CallV1GroupcallHangupOthers(ctx, id)
		}(groupcallID)
	}

	// check the chained calls
	for _, callID := range gc.CallIDs {
		if callID == gc.AnswerCallID {
			continue
		}

		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.reqHandler.CallV1CallHangup(ctx, id)
		}(callID)
	}

	return nil
}

// Hangingup hangs up the groupcalls.
func (h *groupcallHandler) Hangingup(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Hangingup",
		"groupcall_id": id,
	})

	res, err := h.UpdateStatus(ctx, id, groupcall.StatusHangingup)
	if err != nil {
		log.Errorf("Could not update the groupcall status to hangingup. err: %v", err)
		return nil, errors.Wrap(err, "could not update the groupcall status to hangingup.")
	}

	// hanging up the groupcalls
	for _, groupcallID := range res.GroupcallIDs {
		log.Debugf("Hanging up the groupcalls. groupcall_id: %s", groupcallID)
		go func(id uuid.UUID) {
			_, _ = h.reqHandler.CallV1GroupcallHangup(ctx, id)
		}(groupcallID)
	}

	// hanging up the calls
	for _, callID := range res.CallIDs {
		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.reqHandler.CallV1CallHangup(ctx, id)
		}(callID)
	}

	return res, nil
}

// Hangup hangs up the groupcalls.
func (h *groupcallHandler) Hangup(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Hangup",
		"groupcall_id": id,
	})

	res, err := h.UpdateStatus(ctx, id, groupcall.StatusHangup)
	if err != nil {
		log.Errorf("Could not update the groupcall status. err: %v", err)
		return nil, errors.Wrap(err, "could not hangup the groupcall")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallHangup, res)

	if res.MasterGroupcallID != uuid.Nil {
		log.Debugf("Groupcall has master groupcall id. master_groupcall_id: %s", res.MasterGroupcallID)
		go func(id uuid.UUID) {
			if errGroupcall := h.reqHandler.CallV1GroupcallHangupGroupcall(ctx, id); errGroupcall != nil {
				log.Errorf("Could not hangup the related groupcall id from the master_groupcall_id. master_groupcall_id: %s, err: %v", id, errGroupcall)
			}
		}(res.MasterGroupcallID)
	}

	return res, nil
}

// HangupGroupcall handles groupcall's hangup.
func (h *groupcallHandler) HangupGroupcall(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "HangupGroupcall",
		"groupcall_id": id,
	})

	// decrease the groupcall count
	gc, err := h.DecreaseGroupcallCount(ctx, id)
	if err != nil {
		log.Errorf("Could not decrease the groupcall count. err: %v", err)
		return nil, errors.Wrap(err, "could not decrease the call count")
	}

	return h.hangupCommon(ctx, gc)
}

// HangupCall handles call's hangup.
func (h *groupcallHandler) HangupCall(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "HangupCall",
		"groupcall_id": id,
	})

	// decrease the call count
	gc, err := h.DecreaseCallCount(ctx, id)
	if err != nil {
		log.Errorf("Could not decrease the call count. err: %v", err)
		return nil, errors.Wrap(err, "could not decrease the call count")
	}

	return h.hangupCommon(ctx, gc)
}

// hangupCommon handles common hangup process
func (h *groupcallHandler) hangupCommon(ctx context.Context, gc *groupcall.Groupcall) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "hangupCommon",
		"groupcall": gc,
	})

	if gc.GroupcallCount > 0 || gc.CallCount > 0 {
		// groupcall still have ongoing outgoing call. nothing to do here
		return gc, nil
	}

	if gc.AnswerCallID != uuid.Nil {
		// already answered call. nothing to do here
		return h.Hangup(ctx, gc.ID)
	}

	if gc.Status == groupcall.StatusHangingup {
		log.Debugf("The groupcall status is hanging up and all of the outgoing call has hungup. Hangup the groupcall. groupcall_id: %s", gc.ID)
		return h.Hangup(ctx, gc.ID)
	}

	var res *groupcall.Groupcall
	var err error
	switch gc.RingMethod {
	case groupcall.RingMethodRingAll:
		log.Debugf("Groupcall ring method is ring all. And every calls were hungup already. groupcall_id: %s", gc.ID)
		res, err = h.Hangup(ctx, gc.ID)

	case groupcall.RingMethodLinear:
		log.Debugf("Groupcall ring method is linear. groupcall_id: %s", gc.ID)
		res, err = h.hangupRingMethodLinear(ctx, gc)

	default:
		log.Errorf("Unsupported ring method. ring_method: %s", gc.RingMethod)
		res = nil
		err = fmt.Errorf("unsupported ring method")
	}

	if err != nil {
		log.Errorf("Could not handle the call hangup event. err: %v", err)
		return nil, errors.Wrap(err, "could not handle the call hangup event")
	}

	return res, nil
}

// hangupRingMethodLinear handels call hangup for groupcall ring method linear.
// the linear ring method type groupcall needs to make a next call if the groupcall dialing is over.
// this func checks the groupcall's condition and makes a next dialing if it has dialable destination.
func (h *groupcallHandler) hangupRingMethodLinear(ctx context.Context, gc *groupcall.Groupcall) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "hangupRingMethodLinear",
		"groupcall": gc,
	})

	// get master call info and check the status
	if gc.MasterCallID != uuid.Nil {
		c, err := h.reqHandler.CallV1CallGet(ctx, gc.MasterCallID)
		if err != nil {
			log.Errorf("Could not get master call info. err: %v", err)
			return nil, errors.Wrap(err, "could not get master call info.")
		}
		log = log.WithField("master_call", c)
		log.Debugf("Found master call info. master_call_id: %s", c.ID)

		if c.Status == call.StatusHangup || c.Status == call.StatusCanceling || c.Status == call.StatusTerminating {
			log.Infof("The master call status is hangup or being hangingup. No need to make a next dialing. Hangup the groupcall. master_call_id: %s, master_call_status: %s", c.ID, c.Status)
			return h.Hangup(ctx, gc.ID)
		}
	}

	// get  master groupcall info and check the status
	if gc.MasterGroupcallID != uuid.Nil {
		tmp, err := h.Get(ctx, gc.MasterGroupcallID)
		if err != nil {
			log.Errorf("Could not get master groupcall info. err:%v", err)
			return nil, errors.Wrap(err, "could not get master groupcall info")
		}
		log = log.WithField("master_groupcall", tmp)
		log.Debugf("Found master groupcall info. master_groupcall_id: %s", tmp.ID)

		if tmp.Status == groupcall.StatusHangingup || tmp.Status == groupcall.StatusHangup {
			log.Infof("The master groupcall status is hangup or being hangingup. No need to make a next dialing. Hangup the groupcall. master_groupcall_id: %s, master_groupcall_status: %s", tmp.ID, tmp.Status)
			return h.Hangup(ctx, gc.ID)
		}
	}

	// check the dial index
	if gc.DialIndex >= len(gc.Destinations)-1 {
		log.Infof("Already dialed to the all of destinations. No more dialable destination left. dial_index: %d", gc.DialIndex)
		return h.Hangup(ctx, gc.ID)
	}

	// dial to the next destination
	res, err := h.dialNextDestination(ctx, gc)
	if err != nil {
		log.Errorf("Could not dial to the next destination. err: %v", err)
		return nil, errors.Wrap(err, "could not dial to the next destination")
	}

	return res, nil
}
