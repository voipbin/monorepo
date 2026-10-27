package callhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/outboundconfig"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
)

func Test_EventCUCustomerDeleted_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil("7a1b2c3d-0000-4000-8000-00000000c2b1")
	callID := uuid.FromStringOrNil("7a1b2c3d-0000-4000-8000-00000000c2b2")
	configID := uuid.FromStringOrNil("7a1b2c3d-0000-4000-8000-00000000c2b3")
	listErr := fmt.Errorf("list failure")

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr error
	}{
		{
			name: "list error is returned",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(nil, listErr)
			},
			expectErr: listErr,
		},
		{
			name: "call delete failure is skipped and a missing outbound config lookup is only logged",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return([]*call.Call{{Identity: commonidentity.Identity{ID: callID}}}, nil)
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
				m.outbound.EXPECT().GetByCustomerID(ctx, customerID).Return(nil, fmt.Errorf("lookup failure"))
			},
		},
		{
			name: "outbound config delete failure is only logged",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return([]*call.Call{}, nil)
				m.outbound.EXPECT().GetByCustomerID(ctx, customerID).Return(&outboundconfig.OutboundConfig{ID: configID, CustomerID: customerID}, nil)
				m.outbound.EXPECT().Delete(ctx, configID).Return(nil, fmt.Errorf("delete failure"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			err := h.EventCUCustomerDeleted(ctx, &cucustomer.Customer{ID: customerID})
			if tt.expectErr == nil && err != nil {
				t.Errorf("Wrong error. got: %v", err)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_EventCUCustomerFrozen_cov2b(t *testing.T) {
	customerID := uuid.FromStringOrNil("7a1b2c3d-0000-4000-8000-00000000c2b4")
	callID := uuid.FromStringOrNil("7a1b2c3d-0000-4000-8000-00000000c2b5")
	listErr := fmt.Errorf("list failure")

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *cov2bMocks)

		expectErr error
	}{
		{
			name: "list error is returned",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(nil, listErr)
			},
			expectErr: listErr,
		},
		{
			name: "hangup failure of one call is only logged",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return([]*call.Call{{Identity: commonidentity.Identity{ID: callID}}}, nil)
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf("db failure"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			err := h.EventCUCustomerFrozen(ctx, &cucustomer.Customer{ID: customerID})
			if tt.expectErr == nil && err != nil {
				t.Errorf("Wrong error. got: %v", err)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
