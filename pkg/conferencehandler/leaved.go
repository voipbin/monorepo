package conferencehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
)

// leaved handles event the channel has left from the bridge
// when the channel has left from the conference bridge, this func will be fired.
func (h *conferenceHandler) leaved(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()

	log := logrus.WithFields(logrus.Fields{
		"channel_id":      cn.ID,
		"bridge_id":       br.ID,
		"conference_type": br.ReferenceType,
		"conference_id":   br.ReferenceID,
	})

	// get conference info
	cf, err := h.db.ConferenceGet(ctx, br.ReferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}

	switch cf.Type {
	case conference.TypeConnect:
		return h.leavedConferenceTypeConnect(cf, cn, br)

	case conference.TypeConference:
		return h.leavedConferenceTypeConference(cf, cn, br)

	default:
		log.Errorf("Could not find correct event handler.")
		return fmt.Errorf("could not find connrect event handler")
	}
}

// leavedConferenceTypeConnect
func (h *conferenceHandler) leavedConferenceTypeConnect(cf *conference.Conference, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"conference_id": cf.ID,
		"channel_id":    cn.ID,
		"bridge_id":     br.ID,
	})

	if len(br.ChannelIDs) <= 0 {
		if err := h.Destroy(cf.ID); err != nil {
			log.Errorf("Could not destroy the connect type conference. err: %v", err)
			return err
		}
	} else {
		if err := h.Terminate(cf.ID); err != nil {
			log.Errorf("Could not terminate the connect type conference. err: %v", err)
			return err
		}
	}

	return nil
}

// leavedConferenceTypeConference
func (h *conferenceHandler) leavedConferenceTypeConference(cf *conference.Conference, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"conference_id": cf.ID,
		"channel_id":    cn.ID,
		"bridge_id":     br.ID,
	})

	if cf.Status != conference.StatusTerminating {
		// nothing to do here.
		return nil
	}

	if len(br.ChannelIDs) > 0 {
		// we need to wait until all the channel gone
		return nil
	}

	// there's no calls(channels) left in the conference(bridge).
	// let's destroy the conference now
	if err := h.Destroy(cf.ID); err != nil {
		log.Errorf("Could not destroy the conference. err: %v", err)
		return err
	}

	return nil
}
