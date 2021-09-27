package conferencehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

const contextCallJoin = "call-join"

// createEndpointTarget creates target endpoint(destination) address for conference join.
// This will create a SIP destination address towards conference Asterisk to joining the conference.
func (h *conferenceHandler) createEndpointTarget(ctx context.Context, cf *conference.Conference) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"conference": cf.ID,
		},
	)

	// get bridge
	bridge, err := h.db.BridgeGet(ctx, cf.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridgie info. bridge: %s, err: %v", cf.BridgeID, err)
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

	// check the conference bridge exists
	if cf.BridgeID == "" || h.isBridgeExist(ctx, cf.BridgeID) != true {
		// the conference's bridge has gone somehow.
		// we need to create a new bridge for this conference.
		log.Infof("Could not find valid conference bridge. Creating a new bridge for the conference. bridge: %s", cf.BridgeID)

		timeout := time.Second * 1
		if err := h.createConferenceBridgeWithTimeout(ctx, cf.ID, timeout); err != nil {
			log.Errorf("Could not create a conference bridge. err: %v", err)
			return err
		}

		// get updated conference
		cf, err = h.db.ConferenceGet(ctx, conferenceID)
		if err != nil {
			log.Errorf("Could not get conference after bridge update. err: %v", err)
			return err
		}
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
	args := fmt.Sprintf("context=%s,conference_id=%s,bridge_id=%s,call_id=%s",
		contextCallJoin,
		cf.ID.String(),
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

	// set conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, cf.ID); err != nil {
		log.Errorf("Could not set the conference for a call. err: %v", err)

		h.reqHandler.AstChannelHangup(c.AsteriskID, channelID.String(), ari.ChannelCauseNormalClearing)
		return err
	}

	// get updated call
	tmpCall, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get updated call info. err: %v", err)
		h.reqHandler.AstChannelHangup(c.AsteriskID, channelID.String(), ari.ChannelCauseNormalClearing)
		return err
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeCallUpdated, tmpCall.WebhookURI, tmpCall)

	// add the call to conference
	if err := h.db.ConferenceAddCallID(ctx, cf.ID, c.ID); err != nil {
		// we don't kick out the joined call at here.
		// just write log.
		log.Errorf("Could not add the callid into the conference. err: %v", err)
	}
	promConferenceJoinTotal.WithLabelValues(string(cf.Type)).Inc()

	return nil
}

// isBridgeExist returns true if the given bridge does exist
func (h *conferenceHandler) isBridgeExist(ctx context.Context, id string) bool {
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

// createConferenceBridge creates the bridge for conferencing
// func (h *conferenceHandler) createConferenceBridge(ctx context.Context, conferenceType conference.Type, id uuid.UUID) error {
func (h *conferenceHandler) createConferenceBridgeWithTimeout(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	bridgeID := uuid.Must(uuid.NewV4()).String()
	bridgeName := generateBridgeName(bridge.ReferenceTypeConference, id)

	log := logrus.WithFields(
		logrus.Fields{
			"conference": id,
			"bridge":     bridgeID,
		},
	)

	// create a bridge
	if err := h.reqHandler.AstBridgeCreate(requesthandler.AsteriskIDConference, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing}); err != nil {
		log.Errorf("Could not create a bridge for a conference. err: %v", err)
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

	// set new bridge id to the conference
	if err := h.db.ConferenceSetBridgeID(ctx, id, br.ID); err != nil {
		log.Errorf("Could not set the new bridge id for a conference. err: %v", err)
		return err
	}

	return nil
}
