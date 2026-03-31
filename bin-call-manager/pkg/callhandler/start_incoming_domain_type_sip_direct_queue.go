package callhandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIPDirectQueue handles direct hash call routed to a queue resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectQueue(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectQueue",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	q, err := h.reqHandler.QueueV1QueueGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("queue", q).Debugf("Retrieved queue info. queue_id: %s", q.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeTel,
		Target:     q.ID.String(),
		TargetName: q.Name,
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeQueueJoin,
			Option: fmaction.ConvertOption(fmaction.OptionQueueJoin{
				QueueID: q.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, q.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct queue join. queue_id: %s", q.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, q.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
