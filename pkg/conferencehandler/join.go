package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func (h *conferenceHandler) createEndpointTarget(ctx context.Context, cf *conference.Conference) (string, error) {
	// get bridge
	bridge, err := h.db.BridgeGet(ctx, cf.BridgeID)
	if err != nil {
		return "", err
	}

	// get bridge asterisk's address
	address, err := h.cache.AsteriskAddressInternerGet(ctx, bridge.AsteriskID)
	if err != nil {
		return "", err
	}

	res := fmt.Sprintf("PJSIP/conf-join/sip:%s@%s:5060", bridge.ID, address)

	return res, nil
}

// Join handler call's join request.
// 1. Creates a bridge(conference joining type) and put the call's channel into the bridge
// 2. Creates a new channel for joining to the conference.
func (h *conferenceHandler) Join(conferenceID, callID uuid.UUID) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"conference": conferenceID,
			"call":       callID,
		})
	log.Info("Starting to join the call to the conference.")

	// get conference
	cf, err := h.db.ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
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

	// create a joining bridge
	bridgeID := uuid.Must(uuid.NewV4()).String()
	bridgeName := generateBridgeName(conference.TypeNone, conferenceID, true)
	if err := h.reqHandler.AstBridgeCreate(c.AsteriskID, bridgeID, bridgeName, bridge.TypeMixing); err != nil {
		return fmt.Errorf("could not create a bridge for conference joining. err: %v", err)
	}

	// put the call's channel into the bridge
	// put the channel into the bridge
	if err := h.reqHandler.AstBridgeAddChannel(c.AsteriskID, bridgeID, c.ChannelID, "", false, false); err != nil {
		h.reqHandler.AstBridgeDelete(c.AsteriskID, bridgeID)
		return fmt.Errorf("could not add the channel into the the bridge. bridge: %s", bridgeID)
	}

	// create a dial string
	dialDestination, err := h.createEndpointTarget(ctx, cf)
	if err != nil {
		log.Errorf("Could not create a dial destination. err: %v", err)
		return err
	}
	log.Debugf("Created dial destination. destination: %s", dialDestination)

	// create a channel args
	// CONFERENCE_ID: The conference ID which this channel belongs to.
	// BRIDGE_ID: The bridge ID where this channel entered after StasisStart.
	// CALL_ID: The call ID which this channel has related with.
	args := fmt.Sprintf("CONTEXT=%s,CONFERENCE_ID=%s,BRIDGE_ID=%s,CALL_ID=%s",
		contextConferenceJoin,
		cf.ID.String(),
		bridgeID,
		c.ID.String(),
	)

	// create a another channel with joining context
	channelID := uuid.Must(uuid.NewV4())
	if err := h.reqHandler.AstChannelCreate(c.AsteriskID, channelID.String(), args, dialDestination, "", "", ""); err != nil {
		log.Errorf("Could not create a channel for joining. err: %v", err)
		return err
	}

	return nil
}
