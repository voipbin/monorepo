package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Join handles call's join request.
// 1. Creates a bridge(confbridge joining type) and put the call's channel into the bridge
// 2. Creates a new channel for joining to the confbridge.
func (h *confbridgeHandler) Join(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Join",
		"confbridge_id": id,
		"call_id":       callID,
	})
	log.Info("Starting to join the call to the confbridge.")

	// get confbridge
	cb, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge. err: %v", err)
		return err
	}

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "could not get call info")
	}

	// check the confbridge's bridge does exist
	if cb.BridgeID == "" || !h.bridgeHandler.IsExist(ctx, cb.BridgeID) {
		// the conference's bridge has gone somehow.
		// we need to create a new bridge for this conference.
		log.Infof("Could not find valid conference bridge. Creating a new bridge for the conference. bridge: %s", cb.BridgeID)

		cb, err = h.createConfbridgeBridge(ctx, cb.ID)
		if err != nil {
			log.Errorf("Could not create a conference bridge. err: %v", err)
			return err
		}
	}

	// create a dial string
	dialDestination, err := h.createEndpointTarget(ctx, cb)
	if err != nil {
		log.Errorf("Could not create a dial destination. err: %v", err)
		return err
	}
	log.Debugf("Created dial destination. destination: %s", dialDestination)

	// create a channel args
	// confbridge_id: The conference ID which this channel belongs to.
	// bridge_id: The bridge ID where this channel entered after StasisStart.
	// call_id: The call ID which this channel has related with.
	args := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextJoinCall,
		channel.StasisDataTypeConfbridgeID, cb.ID.String(),
		channel.StasisDataTypeBridgeID, c.BridgeID,
		channel.StasisDataTypeCallID, c.ID.String(),
	)

	// create variables
	variables := map[string]string{
		"PJSIP_HEADER(add," + common.SIPHeaderCallID + ")":       c.ID.String(),
		"PJSIP_HEADER(add," + common.SIPHeaderConfbridgeID + ")": cb.ID.String(),
	}

	// create a another channel with joining context
	channelID := h.utilHandler.UUIDCreate()
	tmp, err := h.channelHandler.StartChannelWithBaseChannel(ctx, c.ChannelID, channelID.String(), args, dialDestination, "", "vp8", "", variables)
	if err != nil {
		log.Errorf("Could not create a channel for joining. err: %v", err)
		return err
	}
	log.WithField("channel", tmp).Debugf("Created a join channel for conference joining. channel_id: %s", tmp.ID)

	return nil
}

// createConfbridgeBridge creates the bridge for confbridge
func (h *confbridgeHandler) createConfbridgeBridge(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "createConfbridgeBridge",
		"confbridge_id": id,
	})

	cb, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, errors.Wrap(err, "could not get confbridge info")
	}

	var bridgeTypes []bridge.Type
	switch cb.Type {
	case confbridge.TypeConnect:
		bridgeTypes = []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}

	case confbridge.TypeConference:
		bridgeTypes = []bridge.Type{bridge.TypeMixing}

	default:
		return nil, fmt.Errorf("unsupported confbridge type. confbridge_type: %s", cb.Type)
	}

	bridgeID := h.utilHandler.UUIDCreate().String()
	bridgeName := generateBridgeName(bridge.ReferenceTypeConfbridge, id)
	log.Debugf("Creating a bridge for confbridge. birdge_id: %s, birdge_name: %s", bridgeID, bridgeName)

	// create a bridge
	br, err := h.bridgeHandler.Start(ctx, requesthandler.AsteriskIDConference, bridgeID, bridgeName, bridgeTypes)
	if err != nil {
		log.Errorf("Could not create bridge for a confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not create a bridge for the confbridge")
	}
	log.WithField("bridge", br).Debugf("Created a bridge for confbridge. bridge_id: %s", br.ID)

	// set new bridge id to the confbridge
	res, err := h.UpdateBridgeID(ctx, cb.ID, br.ID)
	if err != nil {
		log.Errorf("Could not update the bridge id. err: %v", err)
		return nil, errors.Wrap(err, "could not update the bridge id")
	}
	log.WithField("confbridge", res).Debugf("Updated conbridge info. confbridge_id: %s", res.ID)

	return res, nil
}

// createEndpointTarget creates target endpoint(destination) address for conference join.
// This will create a SIP destination address towards conference Asterisk to joining the conference.
func (h *confbridgeHandler) createEndpointTarget(ctx context.Context, cb *confbridge.Confbridge) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "createEndpointTarget",
		"confbridge_id": cb.ID,
	})

	// get bridge
	bridge, err := h.bridgeHandler.Get(ctx, cb.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridgie info. bridge: %s, err: %v", cb.BridgeID, err)
		return "", err
	}

	// get bridge asterisk's address
	address, err := h.cache.AsteriskAddressInternalGet(ctx, bridge.AsteriskID)
	if err != nil {
		log.Errorf("Could not get conference Asterisk internal address. asterisk: %s", bridge.AsteriskID)
		return "", err
	}

	res := fmt.Sprintf("PJSIP/conf-join/sip:%s@%s:5060", bridge.ID, address)

	return res, nil
}
