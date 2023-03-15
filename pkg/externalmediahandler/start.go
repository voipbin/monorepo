package externalmediahandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// Start starts the external media processing
func (h *externalMediaHandler) Start(ctx context.Context, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})
	log.Debug("Creating the external media.")

	switch referenceType {
	case externalmedia.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, referenceID, externalHost, encapsulation, transport, connectionType, format, direction)

	case externalmedia.ReferenceTypeConfbridge:
		return h.startReferenceTypeConfbridge(ctx, referenceID, externalHost, encapsulation, transport, connectionType, format, direction)

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// startReferenceTypeCall starts the external media processing
func (h *externalMediaHandler) startReferenceTypeCall(ctx context.Context, callID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "startReferenceTypeCall",
		"call_id": callID,
	})
	log.Debug("Creating the external media for call type.")

	c, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get a call info. err: %v", err)
		return nil, err
	}

	// get channel
	ch, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, err
	}

	// create a bridge
	bridgeID := h.utilHandler.CreateUUID().String()
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", bridge.ReferenceTypeCallSnoop, c.ID)
	br, err := h.bridgeHandler.Start(ctx, ch.AsteriskID, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
	if err != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", err)
		return nil, err
	}

	// create a snoop channel
	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s,bridge_id=%s",
		common.ContextExternalSoop,
		c.ID,
		br.ID,
	)
	snoopID := h.utilHandler.CreateUUID().String()
	tmp, err := h.channelHandler.StartSnoop(ctx, ch.ID, snoopID, appArgs, channel.SnoopDirection(direction), channel.SnoopDirectionBoth)
	if err != nil {
		log.Errorf("Could not create a snoop channel for the external media. error: %v", err)
		return nil, err
	}
	log.WithField("channel", tmp).Debugf("Created a new snoop channel. channel_id: %s", tmp.ID)

	// start external media
	res, err := h.startExternalMedia(ctx, ch.AsteriskID, br.ID, externalmedia.ReferenceTypeCall, c.ID, externalHost)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startReferenceTypeConfbridge starts the external media processing reference type
func (h *externalMediaHandler) startReferenceTypeConfbridge(ctx context.Context, confbridgeID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConfbridge",
		"confbridge_id": confbridgeID,
		"external_host": externalHost,
	})
	log.Debug("Creating the external media for confbridge type.")

	// get confbridge
	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info")
		return nil, err
	}
	log.WithField("confbridge", cb).Debugf("Found confbridge info. confbridge_id: %s", cb.ID)

	// get bridge
	br, err := h.bridgeHandler.Get(ctx, cb.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return nil, err
	}
	log.WithField("bridge", br).Debugf("Found bridge info. bridge_id: %s", br.ID)

	// start external media
	res, err := h.startExternalMedia(ctx, br.AsteriskID, br.ID, externalmedia.ReferenceTypeConfbridge, cb.ID, externalHost)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startExternalMedia starts the external media and create external media database record
func (h *externalMediaHandler) startExternalMedia(ctx context.Context, asteriskID string, bridgeID string, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, externalHost string) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startExternalMedia",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})
	// create a external media channel
	chData := fmt.Sprintf("context=%s,bridge_id=%s,reference_type=%s,reference_id=%s", common.ContextExternalMedia, bridgeID, referenceType, referenceID)
	extChannelID := h.utilHandler.CreateUUID().String()
	extCh, err := h.channelHandler.StartExternalMedia(ctx, asteriskID, extChannelID, externalHost, constEncapsulation, constTransport, constConnectionType, constFormat, constDirection, chData, nil)
	if err != nil {
		log.Errorf("Could not create a external media channel. err: %v", err)
		return nil, err
	}

	// parse local localIP and port
	localIP := ""
	localPort := 0
	if tmp := extCh.Data[ChannelValiableExternalMediaLocalAddress]; tmp != nil {
		localIP = tmp.(string)
	}
	if tmp := extCh.Data[ChannelValiableExternalMediaLocalPort]; tmp != nil {
		localPort, _ = strconv.Atoi(tmp.(string))
	}

	res, err := h.Create(ctx, asteriskID, extChannelID, referenceType, referenceID, localIP, localPort, externalHost, constEncapsulation, constTransport, constConnectionType, constFormat, constDirection)
	if err != nil {
		log.Errorf("Could not create a external media. err: %v", err)
		return nil, err
	}

	return res, nil
}
