package customerhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_UpdateIdentityVerificationStatus(t *testing.T) {
	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	tests := []struct {
		name   string
		status customer.IdentityVerificationStatus

		expectDB  func(m *dbhandler.MockDBHandler)
		expectPub bool
		expectErr bool
	}{
		{
			name:      "invalid status is rejected before any lookup",
			status:    customer.IdentityVerificationStatus("bogus"),
			expectDB:  func(m *dbhandler.MockDBHandler) {},
			expectErr: true,
		},
		{
			name:   "get customer fails",
			status: customer.IdentityVerificationStatusVerified,
			expectDB: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("db down"))
			},
			expectErr: true,
		},
		{
			name:   "same status is a no-op without update or event",
			status: customer.IdentityVerificationStatusVerified,
			expectDB: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: customer.IdentityVerificationStatusVerified,
				}, nil)
			},
		},
		{
			name:   "update fails",
			status: customer.IdentityVerificationStatusPending,
			expectDB: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: customer.IdentityVerificationStatusNone,
				}, nil)
				m.EXPECT().CustomerUpdate(gomock.Any(), id, map[customer.Field]any{
					customer.FieldIdentityVerificationStatus: "pending",
				}).Return(fmt.Errorf("update failed"))
			},
			expectErr: true,
		},
		{
			name:   "re-read after update fails",
			status: customer.IdentityVerificationStatusPending,
			expectDB: func(m *dbhandler.MockDBHandler) {
				first := m.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: customer.IdentityVerificationStatusNone,
				}, nil)
				upd := m.EXPECT().CustomerUpdate(gomock.Any(), id, map[customer.Field]any{
					customer.FieldIdentityVerificationStatus: "pending",
				}).Return(nil).After(first)
				m.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("db down")).After(upd)
			},
			expectErr: true,
		},
		{
			name:   "success publishes the updated customer",
			status: customer.IdentityVerificationStatusRejected,
			expectDB: func(m *dbhandler.MockDBHandler) {
				first := m.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: customer.IdentityVerificationStatusPending,
				}, nil)
				upd := m.EXPECT().CustomerUpdate(gomock.Any(), id, map[customer.Field]any{
					customer.FieldIdentityVerificationStatus: "rejected",
				}).Return(nil).After(first)
				m.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: customer.IdentityVerificationStatusRejected,
				}, nil).After(upd)
			},
			expectPub: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &customerHandler{
				reqHandler:    requesthandler.NewMockRequestHandler(mc),
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			tt.expectDB(mockDB)
			if tt.expectPub {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerIdentityVerificationUpdated, &customer.Customer{
					ID:                         id,
					IdentityVerificationStatus: tt.status,
				}).Return()
			}

			res, err := h.UpdateIdentityVerificationStatus(context.Background(), id, tt.status)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.ID != id || res.IdentityVerificationStatus != tt.status {
				t.Errorf("Wrong match. expect status: %v, got: %v", tt.status, res.IdentityVerificationStatus)
			}
		})
	}
}
