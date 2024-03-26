package externalmediahandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// Start starts the external media processing
func (h *externalMediaHandler) Start(ctx context.Context, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
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
func (h *externalMediaHandler) startReferenceTypeCall(ctx context.Context, callID uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
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

	// start external media
	res, err := h.startExternalMedia(ctx, ch.AsteriskID, c.BridgeID, externalmedia.ReferenceTypeCall, c.ID, externalHost, encapsulation, transport, format)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startReferenceTypeConfbridge starts the external media processing reference type
func (h *externalMediaHandler) startReferenceTypeConfbridge(ctx context.Context, confbridgeID uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error) {
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
	res, err := h.startExternalMedia(ctx, br.AsteriskID, br.ID, externalmedia.ReferenceTypeConfbridge, cb.ID, externalHost, encapsulation, transport, format)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startExternalMedia starts the external media and create external media database record
func (h *externalMediaHandler) startExternalMedia(ctx context.Context, asteriskID string, bridgeID string, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, format string) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startExternalMedia",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	if encapsulation == "" {
		log.Debugf("The requested external media has no encapsulation. Use the default encapsulation. default_encapsulation: %s", defaultEncapsulation)
		encapsulation = defaultEncapsulation
	}

	if transport == "" {
		log.Debugf("The requested transport has no transport. Use the default. default_transport: %s", defaultTransport)
		transport = defaultTransport
	}

	if format == "" {
		log.Debugf("The requested external media has no format. Use the default format. default_encapsulation: %s", defaultFormat)
		format = defaultFormat
	}

	// create a external media channel
	chData := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextExternalMedia,
		channel.StasisDataTypeBridgeID, bridgeID,
		channel.StasisDataTypeReferenceType, referenceType,
		channel.StasisDataTypeReferenceID, referenceID,
	)
	if encapsulation == externalmedia.EncapsulationAudioSocket {
		// because of the audiosocket required to set the channel data as a some random uuid-string,
		// we can not set the key=value pair string.
		// so we are putting the bridge here to put the channel into the bridge easily.
		chData = bridgeID
		log.Debugf("The encapsulation is audiosocket. Use the channel id as the channel data in force. ch_data: %s", chData)
	}

	extChannelID := h.utilHandler.UUIDCreate().String()
	extCh, err := h.channelHandler.StartExternalMedia(ctx, asteriskID, extChannelID, externalHost, string(encapsulation), string(transport), defaultConnectionType, format, defaultDirection, chData, nil)
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

	res, err := h.Create(ctx, asteriskID, extChannelID, referenceType, referenceID, localIP, localPort, externalHost, encapsulation, defaultTransport, defaultConnectionType, defaultFormat, defaultDirection)
	if err != nil {
		log.Errorf("Could not create a external media. err: %v", err)
		return nil, err
	}

	return res, nil
}
