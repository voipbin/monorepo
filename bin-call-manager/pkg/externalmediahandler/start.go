package externalmediahandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/playback"
)

// Start starts the external media processing
func (h *externalMediaHandler) Start(
	ctx context.Context,
	id uuid.UUID,
	referenceType externalmedia.ReferenceType,
	referenceID uuid.UUID,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	transportData string,
	connectionType string,
	format string,
	directionListen externalmedia.Direction,
	directionSpeak externalmedia.Direction,
) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})
	log.Debug("Creating the external media.")

	if id == uuid.Nil {
		log.Debugf("The requested id is empty. Generate a new id.")
		id = h.utilHandler.UUIDCreate()
	}

	promExternalMediaStartTotal.WithLabelValues(string(referenceType), string(encapsulation)).Inc()

	switch referenceType {
	case externalmedia.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, format, directionListen, directionSpeak)

	case externalmedia.ReferenceTypeConfbridge:
		return h.startReferenceTypeConfbridge(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, format)

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// startReferenceTypeCall starts the external media processing
func (h *externalMediaHandler) startReferenceTypeCall(
	ctx context.Context,
	id uuid.UUID,
	callID uuid.UUID,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	transportData string,
	format string,
	directionListen externalmedia.Direction,
	directionSpeak externalmedia.Direction,
) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "startReferenceTypeCall",
		"id":      id,
		"call_id": callID,
	})
	log.Debug("Creating the external media for call type.")

	c, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get a call info. err: %v", err)
		return nil, errors.Wrapf(err, "could not get call info for call_id: %s", callID)
	}

	// get channel
	ch, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, errors.Wrapf(err, "could not get channel info for channel_id: %s", c.ChannelID)
	}

	// create a bridge
	bridgeID := h.utilHandler.UUIDCreate().String()
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", bridge.ReferenceTypeCallSnoop, c.ID)
	br, err := h.bridgeHandler.Start(ctx, ch.AsteriskID, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
	if err != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", err)
		return nil, err
	}

	// set app args for snoop channel
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextExternalSnoop,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeBridgeID, br.ID,
	)

	// create snoop channel
	snoopID := h.utilHandler.UUIDCreate().String()
	tmp, err := h.channelHandler.StartSnoop(ctx, ch.ID, snoopID, appArgs, channel.SnoopDirection(directionListen), channel.SnoopDirection(directionSpeak))
	if err != nil {
		log.Errorf("Could not create a snoop channel for the external media. error: %v", err)
		return nil, err
	}
	log.WithField("snoop_channel", tmp).Debugf("Created a new snoop channel. channel_id: %s", tmp.ID)

	// start silence playback
	// we are playing a silence playback to the call's callbridge.
	// because if we play the silence playback to the call's channel, it blocks other media play on the call's channel.
	playbackID := fmt.Sprintf("%s%s", playback.IDPrefixExternalMedia, id.String())
	if errPlay := h.bridgeHandler.Play(ctx, c.BridgeID, playbackID, []string{defaultSilencePlaybackMedia}, "", 0, 0); errPlay != nil {
		return nil, errors.Wrapf(errPlay, "could not start silence playback for channel_id: %s", ch.ID)
	}
	log.WithField("playback_id", playbackID).Debugf("Started silence playback for the channel. channel_id: %s", ch.ID)

	// start external media
	res, err := h.startExternalMedia(
		ctx,
		id,
		ch.AsteriskID,
		br.ID,
		playbackID,
		externalmedia.ReferenceTypeCall,
		c.ID,
		externalHost,
		encapsulation,
		transport,
		transportData,
		format,
		directionListen,
		directionSpeak,
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startReferenceTypeConfbridge starts the external media processing reference type
func (h *externalMediaHandler) startReferenceTypeConfbridge(
	ctx context.Context,
	id uuid.UUID,
	confbridgeID uuid.UUID,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	transportData string,
	format string,
) (*externalmedia.ExternalMedia, error) {
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
	res, err := h.startExternalMedia(
		ctx,
		id,
		br.AsteriskID,
		br.ID,
		"", // playbackID is not used in confbridge
		externalmedia.ReferenceTypeConfbridge,
		cb.ID,
		externalHost,
		encapsulation,
		transport,
		transportData,
		format,
		externalmedia.DirectionBoth, // confbridge is always both direction
		externalmedia.DirectionBoth, // confbridge is always both direction
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}

	return res, nil
}

// startExternalMedia starts the external media and create external media database record
func (h *externalMediaHandler) startExternalMedia(
	ctx context.Context,
	id uuid.UUID,
	asteriskID string,
	bridgeID string,
	playbackID string,
	referenceType externalmedia.ReferenceType,
	referenceID uuid.UUID,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	transportData string,
	format string,
	directionListen externalmedia.Direction,
	directionSpeak externalmedia.Direction,
) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startExternalMedia",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	if id == uuid.Nil {
		log.Debugf("The requested id is empty. Generate a new id.")
		id = h.utilHandler.UUIDCreate()
	}

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

	var chData string
	if encapsulation == externalmedia.EncapsulationAudioSocket {
		// because of the audiosocket required to set the channel data as a some random uuid-string,
		// we can not set the key=value pair string.
		// so we are putting the bridge here to put the channel into the bridge easily.
		// create a external media channel
		chData = id.String()
		log.Debugf("The encapsulation is audiosocket. Use the channel id as the channel data in force. ch_data: %s", chData)
	} else {
		chData = fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
			channel.StasisDataTypeContextType, channel.ContextTypeCall,
			channel.StasisDataTypeContext, channel.ContextExternalMedia,
			channel.StasisDataTypeBridgeID, bridgeID,
			channel.StasisDataTypeReferenceType, referenceType,
			channel.StasisDataTypeReferenceID, referenceID,
			channel.StasisDataTypeExternalMediaID, id,
		)
	}

	extChannelID := h.utilHandler.UUIDCreate().String()
	em, err := h.Create(
		ctx,
		id,
		asteriskID,
		extChannelID,
		bridgeID,
		playbackID,
		referenceType,
		referenceID,
		"",
		0,
		externalHost,
		encapsulation,
		transport,
		transportData,
		defaultConnectionType,
		format,
		directionListen,
		directionSpeak,
	)
	if err != nil {
		log.Errorf("Could not create a external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Created a new external media")

	extCh, err := h.channelHandler.StartExternalMedia(
		ctx,
		asteriskID,
		extChannelID,
		externalHost,
		string(encapsulation),
		string(transport),
		transportData,
		defaultConnectionType,
		format,
		defaultDirection,
		chData,
		nil,
	)
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

	res, err := h.UpdateLocalAddress(ctx, id, localIP, localPort)
	if err != nil {
		log.Errorf("Could not update the local address. err: %v", err)
		return nil, err
	}

	return res, nil
}
