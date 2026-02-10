package callhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	nmnumber "monorepo/bin-number-manager/models/number"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIP handles sip domain type incoming call.
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIP",
		"channel_id": cn.ID,
	})

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
