package callhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startIncomingDomainTypeSIPDirectQueue(t *testing.T) {
	queueID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")
	tmpFlowID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000003")

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0100"}
	cn := &channel.Channel{ID: "ch-queue-1"}
	d := &dmdirect.Direct{ResourceID: queueID}
	queue := &qmqueue.Queue{
		Identity: commonidentity.Identity{ID: queueID, CustomerID: customerID},
		Name:     "support queue",
	}

	t.Run("queue lookup failure rejects the call with no route", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()

		m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(nil, fmt.Errorf("queue lookup failure"))
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)

		if err := h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d, source); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("flow creation failure rejects the call as a server error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()

		m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(queue, nil)
		m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("flow creation failure"))
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder).Return(nil, nil)

		if err := h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d, source); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("temporary flow answers then joins the queue", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()

		m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(queue, nil)
		m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, _ fmflow.Type, _ string, detail string, actions []fmaction.Action, _ uuid.UUID, _ bool) (*fmflow.Flow, error) {
				if !strings.Contains(detail, queueID.String()) {
					t.Errorf("Wrong detail. expect it to contain the queue id, got: %s", detail)
				}
				if len(actions) != 2 || actions[0].Type != fmaction.TypeAnswer || actions[1].Type != fmaction.TypeQueueJoin {
					t.Fatalf("Wrong actions. expect: answer, queue_join, got: %v", actions)
				}

				var option fmaction.OptionQueueJoin
				if err := fmaction.ParseOption(actions[1].Option, &option); err != nil || option.QueueID != queueID {
					t.Errorf("Wrong queue_join option. expect queue id: %s, got: %v (%v)", queueID, option.QueueID, err)
				}

				return &fmflow.Flow{Identity: commonidentity.Identity{ID: tmpFlowID, CustomerID: customerID}}, nil
			})

		// the flow start rejects a customer that is not active
		m.util.EXPECT().UUIDCreate().Return(uuid.Nil)
		m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{Status: cucustomer.StatusFrozen}, nil)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseCallRejected).Return(nil, nil)

		if err := h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d, source); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}
