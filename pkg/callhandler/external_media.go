package callhandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// list of channel variables
const (
	ChannelValiableExternalMediaLocalPort    = "UNICASTRTP_LOCAL_PORT"
	ChannelValiableExternalMediaLocalAddress = "UNICASTRTP_LOCAL_ADDRESS"
)

// ExternalMediaStart starts the external media processing
func (h *callHandler) ExternalMediaStart(callID uuid.UUID, isCallMedia bool, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": callID,
		},
	)
	log.Debug("Creating the external media.")

	ctx := context.Background()
	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get a call info. err: %v", err)
		return nil, err
	}

	// create a bridge
	bridgeID := uuid.Must(uuid.NewV4())
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", bridge.ReferenceTypeCallSnoop, c.ID)
	if errBridge := h.reqHandler.AstBridgeCreate(c.AsteriskID, bridgeID.String(), bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}); errBridge != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", errBridge)
		return nil, errBridge
	}

	// create a snoop channel
	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s,bridge_id=%s",
		ContextExternalSoop,
		c.ID,
		bridgeID,
	)
	snoopID := uuid.Must(uuid.NewV4())
	if errSnoop := h.reqHandler.AstChannelCreateSnoop(c.AsteriskID, c.ChannelID, snoopID.String(), appArgs, channel.SnoopDirection(direction), channel.SnoopDirectionBoth); errSnoop != nil {
		log.Errorf("Could not create a snoop channel for the external media. error: %v", errSnoop)
		return nil, errSnoop
	}

	// create a external media channel
	// set data
	chData := fmt.Sprintf("context=%s,bridge_id=%s,call_id=%s", ContextExternalMedia, bridgeID.String(), c.ID.String())
	extChannelID := uuid.Must(uuid.NewV4())
	extCh, err := h.reqHandler.AstChannelExternalMedia(c.AsteriskID, extChannelID.String(), externalHost, encapsulation, transport, connectionType, format, direction, chData, nil)
	if err != nil {
		log.Errorf("Could not create a external media channel. err: %v", err)
		return nil, err
	}

	if isCallMedia == false {
		return extCh, nil
	}

	// parse local ip and port
	ip := ""
	port := 0
	if tmp := extCh.Data[ChannelValiableExternalMediaLocalAddress]; tmp != nil {
		ip = tmp.(string)
	}
	if tmp := extCh.Data[ChannelValiableExternalMediaLocalPort]; tmp != nil {
		port, err = strconv.Atoi(tmp.(string))
	}

	extMedia := &externalmedia.ExternalMedia{
		CallID:         callID,
		AsteriskID:     c.AsteriskID,
		ChannelID:      extChannelID.String(),
		LocalIP:        ip,
		LocalPort:      port,
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
	}

	if errDB := h.db.ExternalMediaSet(ctx, callID, extMedia); errDB != nil {
		log.Errorf("Could not set the external media info to the database. err: %v", errDB)
	}

	return extCh, nil
}

// ExternalMediaStop stops the external media processing
func (h *callHandler) ExternalMediaStop(callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": callID,
		},
	)
	log.Debug("Stopping the external media.")

	ctx := context.Background()

	// get external media
	extMedia, err := h.db.ExternalMediaGet(ctx, callID)
	if err != nil || extMedia == nil {
		log.Debug("No external media exist. Nothing to do.")
		return nil
	}

	// hangup the external media channel
	if errHangup := h.reqHandler.AstChannelHangup(extMedia.AsteriskID, extMedia.ChannelID, ari.ChannelCauseNormalClearing); errHangup != nil {
		log.Errorf("Could not hangup the external media. err: %v", errHangup)
		return nil
	}

	// delete external media info
	if errExtDelete := h.db.ExternalMediaDelete(ctx, callID); errExtDelete != nil {
		log.Errorf("Could not delete external media info. err: %v", errExtDelete)
		return nil
	}

	return nil
}
