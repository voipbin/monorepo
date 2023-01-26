package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// StartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *confbridgeHandler) StartContextIncoming(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "StartContextIncoming",
			"channel_id": cn.ID,
		},
	)

	channelID := cn.ID
	data := cn.StasisData
	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
	log = log.WithFields(logrus.Fields{
		"call_id":       data["call_id"],
		"confbridge_id": data["confbridge_id"],
		"source":        source,
		"destination":   destination,
		"data":          data,
	})
	log.Debugf("Executing StartContextIncoming. source: %v, destination: %v, data: %v", source, destination, data)

	// set channel type
	if errSet := h.channelHandler.VariableSet(ctx, channelID, "VB-TYPE", string(channel.TypeConfbridge)); errSet != nil {
		log.Errorf("Could not set channel var. err: %v", errSet)
		return errors.Wrap(errSet, "could not set channel variable")
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
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, destination.Target, channelID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the channel to the bridge. err: %v", err)
		return errors.Wrap(err, "could not add the channel to the bridge")
	}

	switch cb.Type {
	case confbridge.TypeConference:
		return h.startContextIncomingTypeConference(ctx, channelID, data, callID, cb)

	case confbridge.TypeConnect:
		return h.startContextIncomingTypeConnect(ctx, channelID, data, callID, cb)

	default:
		log.Errorf("Unsupported confbridge type. confbridge_type: %s", cb.Type)
		return fmt.Errorf("unsupported confbridge type. confbridge_type: %s", cb.Type)
	}
}

// startContextIncomingTypeConference handles confbridge conference type joining channel
func (h *confbridgeHandler) startContextIncomingTypeConference(ctx context.Context, channelID string, data map[string]string, callID uuid.UUID, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "startContextIncomingTypeConference",
			"channel_id":    channelID,
			"call_id":       callID,
			"confbridge_id": cb.ID,
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	log.Debugf("Answering the conference type confbridge joining channel. call_id: %s", callID)
	if errAnswer := h.channelHandler.Answer(ctx, channelID); errAnswer != nil {
		log.Errorf("Could not answer the channel. err: %v", errAnswer)
		return errors.Wrap(errAnswer, "could not answer the channel")
	}

	return nil
}

// startContextIncomingTypeConnect handles confbridge connect type joining channel
func (h *confbridgeHandler) startContextIncomingTypeConnect(ctx context.Context, channelID string, data map[string]string, callID uuid.UUID, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "startContextIncomingTypeConnect",
			"channel_id":    channelID,
			"call_id":       callID,
			"confbridge_id": cb.ID,
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	if len(cb.ChannelCallIDs) == 0 {
		// if it's the first channel, send a ring
		log.Debugf("First channel. Send a ring. call_id: %s", callID)
		if errRing := h.channelHandler.Ring(ctx, channelID); errRing != nil {
			log.Errorf("Could not ring the channel. err: %v", errRing)
			return errors.Wrap(errRing, "could not ring the channel")
		}

		return nil
	}

	// answer the call. it is safe to answer for answered call.
	log.Debugf("Answering the joining channel. call_id: %s", callID)
	if errAnswer := h.channelHandler.Answer(ctx, channelID); errAnswer != nil {
		log.Errorf("Could not answer the channel. err: %v", errAnswer)
		return errors.Wrap(errAnswer, "could not answer the channel")
	}

	// send answer to the channels in the confbridge.
	for tmpChannelID, callID := range cb.ChannelCallIDs {
		if errAnswer := h.channelHandler.Answer(ctx, tmpChannelID); errAnswer != nil {
			log.Errorf("Could not answer the channel. call_id: %s, channel_id: %s, err: %v", callID, tmpChannelID, errAnswer)
		}
	}

	return nil
}
