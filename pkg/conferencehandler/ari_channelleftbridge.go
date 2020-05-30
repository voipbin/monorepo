package conferencehandler

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func (h *conferenceHandler) ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error {
	logrus.WithFields(
		logrus.Fields{
			"channel":         cn.ID,
			"asterisk_id":     cn.AsteriskID,
			"data":            cn.Data,
			"bridge":          br.ID,
			"conference":      br.ConferenceID,
			"conference_type": br.ConferenceType,
			"conference_join": br.ConferenceJoin,
		}).Debug("The conferencehandler handling the ARIChannelLeftBridge.")

	var err error
	switch {

	// join bridge
	case br.ConferenceJoin == true:
		err = h.ariChannelLeftBridgeConferenceJoin(cn, br)

	// echo bridge
	case br.ConferenceType == conference.TypeEcho:
		err = h.ariChannelLeftBridgeConferenceTypeEcho(cn, br)

	// conference bridge
	case br.ConferenceType == conference.TypeConference:
		err = nil

	default:
		err = fmt.Errorf("could not find a correct handler")
	}

	if err != nil {
		logrus.Errorf("Could not handle the channel correctly. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseInterworking)
	}

	return nil
}

// ariChannelLeftBridgeConferenceJoin handles channel left from conference-join bridge
// - hangup the channel if the left channel is
func (h *conferenceHandler) ariChannelLeftBridgeConferenceJoin(cn *channel.Channel, br *bridge.Bridge) error {
	// clean up the join bridge
	h.removeAllChannelsInBridge(br)

	if cn.GetContextType() == channel.ContextTypeCall {
		h.leaved(cn, br)
	} else {
		// hangup if the channel's conext type is not the call.
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
	}

	// destroy the bridge if no more channels there.
	if len(br.ChannelIDs) == 0 {
		if err := h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"conference": br.ConferenceID,
					"bridge":     br.ID,
				}).Errorf("could not delete the bridge. err: %v", err)
		}
	}

	return nil
}

// ariChannelLeftBridgeConferenceJoin handles channel left from conference-join bridge
// - hangup the channel if the left channel is
func (h *conferenceHandler) ariChannelLeftBridgeConferenceTypeEcho(cn *channel.Channel, br *bridge.Bridge) error {
	// clean up the join bridge
	h.removeAllChannelsInBridge(br)

	if cn.GetContextType() == channel.ContextTypeCall {
		if err := h.leaved(cn, br); err != nil {
			h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		}
	} else {
		// hangup if the channel's conext type is not the call.
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
	}

	// destroy the bridge if no more channels there.
	if len(br.ChannelIDs) == 0 {
		if err := h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"conference": br.ConferenceID,
					"bridge":     br.ID,
				}).Errorf("could not delete the bridge. err: %v", err)
		}
	}

	return nil
}
