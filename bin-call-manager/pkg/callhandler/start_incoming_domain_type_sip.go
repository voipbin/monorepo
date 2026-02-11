package callhandler

import (
	"context"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	nmnumber "monorepo/bin-number-manager/models/number"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// directExtensionPrefix is the prefix used for direct extension destinations.
const directExtensionPrefix = "direct."

// startIncomingDomainTypeSIP handles sip domain type incoming call.
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIP",
		"channel_id": cn.ID,
	})

	// check for direct extension hash
	if strings.HasPrefix(cn.DestinationNumber, directExtensionPrefix) {
		hash := strings.TrimPrefix(cn.DestinationNumber, directExtensionPrefix)
		return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, hash)
	}

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
	log = log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the sip incoming call handler. source_target: %s, destination_target: %s", source.Target, destination.Target)

	// get number info
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  destination.Target,
		nmnumber.FieldDeleted: false,
	}
	numbs, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not get numbers info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	if len(numbs) == 0 {
		log.Errorf("No number info found. len: %d", len(numbs))
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	numb := numbs[0]
	log.WithField("number", numb).Infof("Found number info. number_id: %s", numb.ID)

	if numb.CallFlowID == uuid.Nil {
		log.Errorf("Number has no call flow configured. number_id: %s", numb.ID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination)
	return nil
}

// startIncomingDomainTypeSIPDirectExtension handles incoming call to a direct extension via sip:direct.<hash>@sip.voipbin.net.
func (h *callHandler) startIncomingDomainTypeSIPDirectExtension(ctx context.Context, cn *channel.Channel, hash string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIPDirectExtension",
		"channel_id": cn.ID,
		"hash":       hash,
	})
	log.Debugf("Starting direct extension call handler. hash: %s", hash)

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)

	// resolve hash to extension
	ext, err := h.reqHandler.RegistrarV1ExtensionGetByDirectHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get extension by direct hash. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("extension", ext).Debugf("Retrieved extension info. extension_id: %s", ext.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     ext.ID.String(),
		TargetName: ext.Extension,
	}

	// create temp connect flow
	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConnect,
			Option: fmaction.ConvertOption(fmaction.OptionConnect{
				Source: *source,
				Destinations: []commonaddress.Address{
					*destination,
				},
				EarlyMedia:  false,
				RelayReason: false,
			}),
		},
	}

	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		ext.CustomerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for direct extension dialing",
		actions,
		uuid.Nil,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, ext.CustomerID, f.ID, source, destination)

	return nil
}
