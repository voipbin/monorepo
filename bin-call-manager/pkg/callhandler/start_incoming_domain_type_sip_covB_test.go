package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startIncomingDomainTypeSIPDirectConference_covB(t *testing.T) {
	confID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")
	custID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")
	tmpFlowID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000003")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}
	cn := &channel.Channel{ID: "ch-1"}
	d := &dmdirect.Direct{ResourceID: confID}

	tests := []struct {
		name    string
		confErr error
		flowErr error
		cause   ari.ChannelCause
	}{
		{"conference get error", fmt.Errorf("err"), nil, ari.ChannelCauseNoRouteDestination},
		{"flow create error", nil, fmt.Errorf("err"), ari.ChannelCauseNetworkOutOfOrder},
		{"customer not active after flow created", nil, nil, ari.ChannelCauseCallRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			if tt.confErr != nil {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(nil, tt.confErr)
			} else {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(&cfconference.Conference{Identity: commonidentity.Identity{ID: confID, CustomerID: custID}}, nil)
				if tt.flowErr != nil {
					m.req.EXPECT().FlowV1FlowCreate(ctx, custID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, tt.flowErr)
				} else {
					m.req.EXPECT().FlowV1FlowCreate(ctx, custID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tmpFlowID}}, nil)
					m.util.EXPECT().UUIDCreate().Return(uuid.Nil)
					m.req.EXPECT().CustomerV1CustomerGet(ctx, custID).Return(&cucustomer.Customer{Status: cucustomer.StatusFrozen}, nil)
				}
			}
			m.channel.EXPECT().HangingUp(ctx, cn.ID, tt.cause).Return(nil, nil)

			if err := h.startIncomingDomainTypeSIPDirectConference(ctx, cn, d, source); err != nil {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
