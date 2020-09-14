package conferencehandler

import (
	"context"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
)

func (h *conferenceHandler) joined(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"conference":      br.ConferenceID,
			"conference_type": br.ConferenceType,
			"conference_join": br.ConferenceJoin,
			"bridge":          br.ID,
			"channel":         cn.ID,
		})
	log.Debug("The call has joined into the conference.")

	// get call
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		log.Errorf("Could not find a call for channel. err: %v", err)

		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}
	log = log.WithFields(
		logrus.Fields{
			"call": c.ID,
		},
	)

	// set conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, br.ConferenceID); err != nil {
		log.Errorf("Could not set the conference for a call. err: %v", err)

		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// add the call to conference
	if err := h.db.ConferenceAddCallID(ctx, br.ConferenceID, c.ID); err != nil {
		// we don't kick out the joined call at here.
		// just write log.
		log.Errorf("Could not add the callid into the conference. err: %v", err)
	}

	promConferenceJoinTotal.WithLabelValues(string(br.ConferenceType)).Inc()

	return nil
}
