package callhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIPDirectFlow handles direct hash call routed to a flow resource.
// Unlike other resource types, the flow already defines the complete action sequence,
// so no temporary flow is created — it delegates directly to startCallTypeFlow.
func (h *callHandler) startIncomingDomainTypeSIPDirectFlow(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectFlow",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	// get flow info to validate it exists and get customer_id
	f, err := h.reqHandler.FlowV1FlowGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("flow", f).Debugf("Retrieved flow info. flow_id: %s", f.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: d.ResourceID.String(),
	}

	h.startCallTypeFlow(ctx, cn, f.CustomerID, f.ID, source, destination, nil)
	return nil
}
