package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// leaved handles event the channel has left from the bridge
// when the channel has left from the bridge, we have to check below 3 things in respetively.
// becuase if we handle these together, the code will be messed up.
// channel
// bridge
// conference
func (h *conferenceHandler) leaved(cn *channel.Channel, br *bridge.Bridge) error {

	// channel handle
	if err := h.leavedChannel(cn, br); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseInterworking)
	}

	// bridge handle
	h.leavedBridge(cn, br)

	// conference handle
	h.leavedConference(cn, br)

	return nil
}

func (h *conferenceHandler) leavedChannel(cn *channel.Channel, br *bridge.Bridge) error {

	switch cn.GetContextType() {
	// conference
	case channel.ContextTypeConference:
		return h.leavedChannelConf(cn, br)

	// call
	case channel.ContextTypeCall:
		return h.leavedChannelCall(cn, br)
	}

	return nil
}

func (h *conferenceHandler) leavedChannelConf(cn *channel.Channel, br *bridge.Bridge) error {
	switch cn.GetContext() {

	case contextConferenceIncoming:
		// nothing to do
		return nil

	case contextConferenceEcho, contextConferenceJoin:
		h.removeAllChannelsInBridge(br)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return nil

	default:
		return fmt.Errorf("could not find context handler. asterisk: %s, channel: %s, bridge: %s", cn.AsteriskID, cn.ID, br.ID)

	}
}

// leavedChannelCall handle
func (h *conferenceHandler) leavedChannelCall(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"conference":      br.ConferenceID,
			"conference_type": br.ConferenceType,
			"conference_join": br.ConferenceJoin,
			"channel":         cn.ID,
			"bridge":          br.ID,
		},
	)

	// get call info
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// add the call id to the log
	log = log.WithFields(
		logrus.Fields{
			"call": c.ID,
		},
	)

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, uuid.Nil); err != nil {
		log.Errorf("Could not reset the conference for a call. err: %v", err)
		return err
	}
	log.Debug("The call has been leaved from the conference.")

	// remove the call from the conference
	if err := h.db.ConferenceRemoveCallID(ctx, br.ConferenceID, c.ID); err != nil {
		log.Errorf("Could not remove the call id from the conference. err: %v", err)
		return err
	}
	promConferenceLeaveTotal.WithLabelValues(string(br.ConferenceType)).Inc()

	// send a call action next
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		log.Debugf("Could not send the call action next request. err: %v", err)
		return err
	}

	return nil
}

func (h *conferenceHandler) leavedBridge(cn *channel.Channel, br *bridge.Bridge) {
	if br.ConferenceJoin == false {
		return
	}

	if len(br.ChannelIDs) > 0 {
		return
	}

	if err := h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID); err != nil {
		logrus.WithFields(
			logrus.Fields{
				"conference": br.ConferenceID,
				"bridge":     br.ID,
			}).Errorf("could not delete the bridge. err: %v", err)
	}

	return
}

func (h *conferenceHandler) leavedConference(cn *channel.Channel, br *bridge.Bridge) {
	if cn.GetContextType() != channel.ContextTypeCall {
		// nothing to do here
		return
	}

	if h.isTerminatable(context.Background(), br.ConferenceID) == false {
		// the conference is not finished yet.
		return
	}

	h.Terminate(br.ConferenceID)
}

// isTerminatable returns true if the given conference is terminatable
// return false if it gets error
func (h *conferenceHandler) isTerminatable(ctx context.Context, id uuid.UUID) bool {
	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Warningf("Could not get conference after remove the call. err: %v", err)
		return false
	}

	// check there's more calls or not
	switch cf.Type {
	case conference.TypeEcho:
		if len(cf.CallIDs) <= 0 {
			return true
		}

	case conference.TypeConference:
		return false

	default:
		return true
	}

	return false
}
