package callhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// updateStatusRinging updates the call's status to ringing
func (h *callHandler) updateStatusRinging(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "updateStatusRinging",
		"call_id":    c.ID,
		"old_status": c.Status,
		"new_status": call.StatusRinging,
	})

	// check status is updatable
	if !call.IsUpdatableStatus(c.Status, call.StatusRinging) {
		log.Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusRinging)
	}

	res, err := h.UpdateStatus(ctx, c.ID, call.StatusRinging)
	if err != nil {
		log.Errorf("Could not update the call status. err: %v", err)
		return err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallRinging, res)

	return nil
}

// updateStatusProgressing updates the call's status to progressing and does required actions.
func (h *callHandler) updateStatusProgressing(ctx context.Context, cn *channel.Channel, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "updateStatusProgressing",
		"call_id":    c.ID,
		"old_status": c.Status,
		"new_status": call.StatusProgressing,
	})

	// check status is updatable
	if !call.IsUpdatableStatus(c.Status, call.StatusProgressing) {
		log.Infof("The status is not updatable.")
		return fmt.Errorf("status change is not possible. call: %s, old_status: %s, new_status: %s", c.ID, c.Status, call.StatusProgressing)
	}

	res, err := h.UpdateStatus(ctx, c.ID, call.StatusProgressing)
	if err != nil {
		log.Errorf("Could not update the call status. err: %v", err)
		return err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallAnswered, res)

	if c.Direction == call.DirectionIncoming {
		// nothing to do with incoming call at here.
		return nil
	}

	go h.handleSIPCallID(ctx, cn, c)

	return h.ActionNext(ctx, res)
}

// handleSIPCallID gets the sip call id and sets to the VB-SIP_CALLID.
// valid only for outgoing call.
func (h *callHandler) handleSIPCallID(ctx context.Context, cn *channel.Channel, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "handleSIPCallID",
		"call_id":     c.ID,
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})

	sipCallID, err := h.reqHandler.AstChannelVariableGet(ctx, cn.AsteriskID, cn.ID, `CHANNEL(pjsip,call-id)`)
	if err != nil {
		log.Errorf("Could not get channel variable. err: %v", err)
		return
	}
	log.Debugf("Received sip call id. sip_call_id: %s", sipCallID)

	if errSet := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-SIP_CALLID", sipCallID); errSet != nil {
		log.Errorf("Could not set sip_call_id. err: %v", errSet)
		return
	}
}
