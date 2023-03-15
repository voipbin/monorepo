package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// StartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *confbridgeHandler) StartContextIncoming(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "StartContextIncoming",
		"channel_id": cn.ID,
	})

	channelID := cn.ID
	data := cn.StasisData
	log = log.WithFields(logrus.Fields{
		"call_id":       data["call_id"],
		"confbridge_id": data["confbridge_id"],
	})
	log.Debugf("Executing StartContextIncoming. data: %v", data)

	// set channel type
	if errSet := h.channelHandler.SetType(ctx, channelID, channel.TypeConfbridge); errSet != nil {
		log.Errorf("Could not set channel type confbridge. err: %v", errSet)
		return errors.Wrap(errSet, "Could not set channel type confbridge.")
	}

	// get conf info
	confbridgeID := uuid.FromStringOrNil(data["confbridge_id"])
	callID := uuid.FromStringOrNil(data["call_id"])
	if confbridgeID == uuid.Nil || callID == uuid.Nil {
		log.Errorf("Could not get confbridge or call info. confbridge_id: %s, call_id: %s", confbridgeID, callID)
		return fmt.Errorf("could not get confbridge id or call id info")
	}
	log.Debugf("Joining to the confbridge. call_id: %s, confbridge_id: %s", callID, confbridgeID)

	// get confbridge
	cb, err := h.Get(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge. err: %v", err)
		return errors.Wrap(err, "could not get confbridge")
	}
	log.WithField("confbridge", cb).Debugf("Found confbridge. confbridge_id: %s", cb.ID)

	// add the channel to the bridge
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, cb.BridgeID, channelID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the channel to the bridge. err: %v", err)
		return errors.Wrap(err, "could not add the channel to the bridge")
	}

	return nil
}
