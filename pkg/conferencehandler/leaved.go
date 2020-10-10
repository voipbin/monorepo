package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
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

	switch cn.Type {
	case channel.TypeCall:
		return h.leavedChannelCall(cn, br)

	case channel.TypeConf, channel.TypeJoin:
		return h.leavedChannelConf(cn, br)

	default:
		logrus.Warnf("Could not find correct event handler. channel: %s, bridge: %s, type: %s", cn.ID, br.ID, cn.Type)
	}

	return nil
}

func (h *conferenceHandler) leavedChannelConf(cn *channel.Channel, br *bridge.Bridge) error {

	switch cn.Type {
	case channel.TypeConf:
		// nothing todo
		return nil

	case channel.TypeJoin:
		h.removeAllChannelsInBridge(br)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return nil

	default:
		logrus.Warnf("Could not find correct event handler. channel: %s, bridge: %s, type: %s", cn.ID, br.ID, cn.Type)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
	}

	return nil
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

	// remove all other channel in the same bridge
	h.removeAllChannelsInBridge(br)

	// get call info
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
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
	if br.ConferenceJoin != true {
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
	log := logrus.WithFields(
		logrus.Fields{
			"conference": br.ConferenceID,
			"bridge":     br.ID,
			"asterisk":   br.AsteriskID,
		},
	)

	if cn.Type != channel.TypeCall {
		// nothing to do here
		return
	}

	if h.isTerminatable(context.Background(), br.ConferenceID) == false {
		// the conference is not finished yet.
		return
	}
	log.Debug("The conference is terminatable.")

	// send the conference termination request
	if err := h.reqHandler.CallConferenceTerminate(br.ConferenceID, "normal terminating", requesthandler.DelayNow); err != nil {
		log.Errorf("Could not send the conference terminate request. err: %v", err)
	}
}

// isTerminatable returns true if the given conference is terminatable
// return false if it gets error
func (h *conferenceHandler) isTerminatable(ctx context.Context, id uuid.UUID) bool {
	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get conference info. conference: %s, err: %v", id, err)
		return false
	}

	// the conference type's conference will be destroyed when the timeout expired.
	// the other type's conference will be destroyed if the joined calls are lesser than 2
	if cf.Type == conference.TypeConference || len(cf.CallIDs) > 1 {
		return false
	}

	return true
}
