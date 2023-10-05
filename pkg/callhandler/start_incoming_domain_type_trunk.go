package callhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
)

// startIncomingDomainTypeTrunk handles sip incoming doamin type.
func (h *callHandler) startIncomingDomainTypeTrunk(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "startIncomingDomainTypeTrunk",
		"channel": cn,
	})

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)

	log = log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	// get trunk info
	domainName := strings.TrimSuffix(cn.StasisData[channel.StasisDataTypeDomain], common.DomainTrunkSuffix)
	trunk, err := h.reqHandler.RegistrarV1TrunkGetByDomainName(ctx, domainName)
	if err != nil {
		log.Errorf("Could not get trunk info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("trunk", trunk).Debugf("Found trunk info. trunk_id: %s", trunk.ID)

	return h.startIncomingDomainTypeTrunkDestinationTypeTel(ctx, cn, trunk.CustomerID, source, destination)
}

// startIncomingDomainTypeTrunkDestinationTypeTel handles incoming call.
// SIP doamin type and destination type is tel.
func (h *callHandler) startIncomingDomainTypeTrunkDestinationTypeTel(
	ctx context.Context,
	cn *channel.Channel,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeTrunkDestinationTypeTel",
		"channel_id":  cn.ID,
		"customer_id": customerID,
		"source":      source,
		"destination": destination,
	})

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			*destination,
		},
		EarlyMedia:  true,
		RelayReason: true,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: optionData,
		},
	}

	// create tmp flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		customerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for outgoing call dialing",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination)

	return nil
}
