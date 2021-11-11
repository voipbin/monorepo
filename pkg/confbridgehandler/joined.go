package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler/models/event"
)

// Joined handles joined call
func (h *confbridgeHandler) Joined(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Joined",
			"confbridge_id": cn.StasisData["confbridge_id"],
			"call_id":       cn.StasisData["call_id"],
			"conference_id": cn.StasisData["conference_id"],
			"channel_id":    cn.ID,
			"bridge_id":     br.ID,
		},
	)

	confbridgeID := uuid.FromStringOrNil(cn.StasisData["confbridge_id"])
	conferenceID := uuid.FromStringOrNil(cn.StasisData["conference_id"])
	callID := uuid.FromStringOrNil(cn.StasisData["call_id"])

	// add the call/channel info to the confbridge
	if errChannelCallID := h.db.ConfbridgeAddChannelCallID(ctx, confbridgeID, cn.ID, callID); errChannelCallID != nil {
		log.Errorf("Could not add the channel/call info to the confbridge. err: %v", errChannelCallID)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return errors.Wrap(errChannelCallID, "could not set the confbridge's channel/call info")
	}

	// set conference id to the call
	// note: here we are setting the conference's id to the call.
	// we don't set the confbridge id to the call.
	if err := h.db.CallSetConferenceID(ctx, callID, conferenceID); err != nil {
		log.Errorf("Could not set the conference id for a call. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// Publish the event
	evt := &event.ConfbridgeJoinedLeaved{
		ID:           confbridgeID,
		ConferenceID: conferenceID,
		CallID:       callID,
	}
	h.notifyHandler.PublishEvent(ctx, notifyhandler.EventTypeConfbridgeJoined, evt)

	// get updated call info and notify
	call, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get updated call info. But we are keep moving. err: %v", err)
	}
	h.notifyHandler.NotifyEvent(ctx, notifyhandler.EventTypeCallUpdated, call.WebhookURI, call)

	return nil
}
