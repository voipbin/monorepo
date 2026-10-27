package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_EventCUCustomerDeleted_cov2(t *testing.T) {
	ctx := context.Background()

	cu := &cucustomer.Customer{ID: cov2CallID}
	expectFilters := map[confbridge.Field]any{
		confbridge.Field("customer_id"): cov2CallID.String(),
		confbridge.Field("deleted"):     "false",
	}

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectErr bool
	}{
		{
			name: "list failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeList(ctx, uint64(1000), "", expectFilters).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "delete failure of an item is skipped",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeList(ctx, uint64(1000), "", expectFilters).Return([]*confbridge.Confbridge{cov2CB(nil)}, nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.EventCUCustomerDeleted(ctx, cu)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}
