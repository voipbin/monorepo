package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startIncomingDomainTypeSIPDirectFlow_covB(t *testing.T) {
	flowID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	custID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}
	cn := &channel.Channel{ID: "ch-1"}
	d := &dmdirect.Direct{ResourceID: flowID}

	tests := []struct {
		name    string
		flowErr error
		cause   ari.ChannelCause
	}{
		{"flow get error", fmt.Errorf("err"), ari.ChannelCauseNoRouteDestination},
		{"customer not active", nil, ari.ChannelCauseNetworkOutOfOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			if tt.flowErr != nil {
				m.req.EXPECT().FlowV1FlowGet(ctx, flowID).Return(nil, tt.flowErr)
			} else {
				m.req.EXPECT().FlowV1FlowGet(ctx, flowID).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID, CustomerID: custID}}, nil)
				m.util.EXPECT().UUIDCreate().Return(uuid.Nil)
				m.req.EXPECT().CustomerV1CustomerGet(ctx, custID).Return(&cucustomer.Customer{Status: cucustomer.StatusFrozen}, nil)
			}
			m.channel.EXPECT().HangingUp(ctx, cn.ID, tt.cause).Return(nil, nil)

			if err := h.startIncomingDomainTypeSIPDirectFlow(ctx, cn, d, source); err != nil {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
