package confbridgehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

const contextCallJoin = "call-join"

// createEndpointTarget creates target endpoint(destination) address for conference join.
// This will create a SIP destination address towards conference Asterisk to joining the conference.
func (h *confbridgeHandler) createEndpointTarget(ctx context.Context, cb *confbridge.Confbridge) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"confbridge_id": cb.ID,
		},
	)

	// get bridge
	bridge, err := h.db.BridgeGet(ctx, cb.BridgeID)
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

// Join handles call's join request.
// 1. Creates a bridge(confbridge joining type) and put the call's channel into the bridge
// 2. Creates a new channel for joining to the confbridge.
func (h *confbridgeHandler) Join(confbridgeID, callID uuid.UUID) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Join",
			"confbridge_id": confbridgeID.String(),
			"call_id":       callID.String(),
		})
	log.Info("Starting to join the call to the confbridge.")

	// get confbridge
	cb, err := h.db.ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge. err: %v", err)
		return err
	}

	// get call
	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// answer the call. it is safe to call this for answered call.
	if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		log.Errorf("Could not answer the call. err: %v", err)
		return err
	}

	// check the confbridge's bridge does exist
	if cb.BridgeID == "" || !h.isBridgeExist(ctx, cb.BridgeID) {
		// the conference's bridge has gone somehow.
		// we need to create a new bridge for this conference.
		log.Infof("Could not find valid conference bridge. Creating a new bridge for the conference. bridge: %s", cb.BridgeID)

		timeout := time.Second * 1
		if err := h.createConfbridgeBridgeWithTimeout(ctx, cb.ID, timeout); err != nil {
			log.Errorf("Could not create a conference bridge. err: %v", err)
			return err
		}

		// get updated confbridge
		cb, err = h.db.ConfbridgeGet(ctx, confbridgeID)
		if err != nil {
			log.Errorf("Could not get conference after bridge update. err: %v", err)
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
	args := fmt.Sprintf("context=%s,confbridge_id=%s,bridge_id=%s,call_id=%s",
		contextCallJoin,
		cb.ID.String(),
		c.BridgeID,
		c.ID.String(),
	)

	// create a another channel with joining context
	channelID := uuid.Must(uuid.NewV4())
	if err := h.reqHandler.AstChannelCreate(c.AsteriskID, channelID.String(), args, dialDestination, "", "vp8", "", nil); err != nil {
		log.Errorf("Could not create a channel for joining. err: %v", err)
		return err
	}
	log.Debugf("Created a join channel for conference joining. id: %s", channelID)

	return nil
}

// isBridgeExist returns true if the given bridge does exist
func (h *confbridgeHandler) isBridgeExist(ctx context.Context, id string) bool {
	if id == "" {
		logrus.Debugf("The bridge id is invalid. Consider this bridge does not exist. bridge: %s", id)
		return false
	}

	br, err := h.db.BridgeGet(ctx, id)
	if err != nil {
		logrus.Debugf("Could not get bridge info from the database. Consider this bridge does not exist. bridge: %s, err: %v", id, err)
		return false
	}

	if br.TMDelete < defaultTimeStamp {
		logrus.WithFields(
			logrus.Fields{
				"bridge": br,
			},
		).Debugf("The bridge info marked as deleted. Consider this bridge does not exist. bridge: %s", id)
		return false
	}

	// get bridge from the asterisk
	_, err = h.reqHandler.AstBridgeGet(br.AsteriskID, br.ID)
	if err != nil {
		logrus.Debugf("Could not get bridge info from the asterisk. Consider this bridge does not exist. bridge: %s, err: %v", id, err)
		return false
	}

	return true
}

// createConfbridgeBridgeWithTimeout creates the bridge for confbridge
func (h *confbridgeHandler) createConfbridgeBridgeWithTimeout(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	bridgeID := uuid.Must(uuid.NewV4()).String()
	bridgeName := generateBridgeName(bridge.ReferenceTypeConfbridge, id)

	log := logrus.WithFields(
		logrus.Fields{
			"confbridge_id": id,
			"bridge_id":     bridgeID,
		},
	)

	// create a bridge
	if err := h.reqHandler.AstBridgeCreate(requesthandler.AsteriskIDConference, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing}); err != nil {
		log.Errorf("Could not create a bridge for a confbridge. err: %v", err)
		return err
	}

	// set timeout
	tmpCTX, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// get created bridge info within timeout
	br, err := h.db.BridgeGetUntilTimeout(tmpCTX, bridgeID)
	if err != nil {
		log.Errorf("Could not get a created bridge within timeout. err: %v", err)
		return err
	}

	// set new bridge id to the confbridge
	if err := h.db.ConfbridgeSetBridgeID(ctx, id, br.ID); err != nil {
		log.Errorf("Could not set the new bridge id for a confbridge. err: %v", err)
		return err
	}

	return nil
}
