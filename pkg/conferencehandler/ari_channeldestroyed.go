package conferencehandler

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

func (h *conferenceHandler) ARIChannelDestroyed(cn *channel.Channel) error {
	logrus.WithFields(
		logrus.Fields{
			"channel":     cn.ID,
			"asterisk_id": cn.AsteriskID,
			"data":        cn.Data,
		}).Debug("The conferencehandler handling the ChannelDestroyed.")

	switch cn.Data["CONTEXT"] {
	case contextConferenceJoin:
		return h.ariChannelDestroyedContextJoin(cn)
	default:
		return nil
	}
}

func (h *conferenceHandler) ariChannelDestroyedContextJoin(cn *channel.Channel) error {

	// get other channel in the join bridge

	// hangup

	// cn.HangupCause
	return nil
}
