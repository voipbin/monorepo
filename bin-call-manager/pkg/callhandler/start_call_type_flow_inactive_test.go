package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// Test_startCallTypeFlow_inactiveCustomer verifies that a customer whose status does not allow
// incoming calls is rejected with ChannelCauseCallRejected (SIP 403) before any call, activeflow
// or balance work happens. newCovBHandler creates every mock, but only req, util and channel have
// expectations, so any other call (call create, activeflow create, balance check) fails the test.
func Test_startCallTypeFlow_inactiveCustomer(t *testing.T) {
	customerID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")

	tests := []struct {
		name   string
		status cucustomer.Status
	}{
		{"frozen", cucustomer.StatusFrozen},
		{"deleted", cucustomer.StatusDeleted},
		{"expired", cucustomer.StatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			cn := &channel.Channel{ID: "ch-1"}

			m.util.EXPECT().UUIDCreate().Return(uuid.Nil)
			m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{Status: tt.status}, nil)
			m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseCallRejected).Return(nil, nil)

			// source, destination and num are unused: the branch returns before they are read.
			h.startCallTypeFlow(ctx, cn, customerID, flowID, nil, nil, nil)
		})
	}
}
