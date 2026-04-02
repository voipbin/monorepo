package callhandler

import (
	"context"
	"fmt"
	"testing"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ValidateCustomerStatusOutgoing(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID

		responseCustomer *cucustomer.Customer
		responseErr      error

		expectCustomer *cucustomer.Customer
		expectValid    bool
	}{
		{
			name:       "active - allowed",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			expectValid: true,
		},
		{
			name:       "initial - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			expectValid: false,
		},
		{
			name:       "frozen - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			expectValid: false,
		},
		{
			name:       "expired - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			expectValid: false,
		},
		{
			name:       "deleted - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			expectValid: false,
		},
		{
			name:             "customer-manager unavailable - fail open",
			customerID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000006"),
			responseCustomer: nil,
			responseErr:      fmt.Errorf("connection refused"),
			expectCustomer:   nil,
			expectValid:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:  mockReq,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, tt.responseErr)

			cu, valid := h.ValidateCustomerStatusOutgoing(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("ValidateCustomerStatusOutgoing() valid = %v, want %v", valid, tt.expectValid)
			}

			if tt.expectCustomer == nil {
				if cu != nil {
					t.Errorf("ValidateCustomerStatusOutgoing() customer = %v, want nil", cu)
				}
			} else {
				if cu == nil {
					t.Errorf("ValidateCustomerStatusOutgoing() customer = nil, want %v", tt.expectCustomer)
				} else if cu.ID != tt.expectCustomer.ID {
					t.Errorf("ValidateCustomerStatusOutgoing() customer.ID = %v, want %v", cu.ID, tt.expectCustomer.ID)
				}
			}
		})
	}
}

func Test_ValidateCustomerStatusIncoming(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID

		responseCustomer *cucustomer.Customer
		responseErr      error

		expectCustomer *cucustomer.Customer
		expectValid    bool
	}{
		{
			name:       "active - allowed",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			expectValid: true,
		},
		{
			name:       "initial - allowed",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			expectValid: true,
		},
		{
			name:       "frozen - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			expectValid: false,
		},
		{
			name:       "expired - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			expectValid: false,
		},
		{
			name:       "deleted - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			expectValid: false,
		},
		{
			name:             "customer-manager unavailable - fail open",
			customerID:       uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000006"),
			responseCustomer: nil,
			responseErr:      fmt.Errorf("connection refused"),
			expectCustomer:   nil,
			expectValid:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:  mockReq,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, tt.responseErr)

			cu, valid := h.ValidateCustomerStatusIncoming(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("ValidateCustomerStatusIncoming() valid = %v, want %v", valid, tt.expectValid)
			}

			if tt.expectCustomer == nil {
				if cu != nil {
					t.Errorf("ValidateCustomerStatusIncoming() customer = %v, want nil", cu)
				}
			} else {
				if cu == nil {
					t.Errorf("ValidateCustomerStatusIncoming() customer = nil, want %v", tt.expectCustomer)
				} else if cu.ID != tt.expectCustomer.ID {
					t.Errorf("ValidateCustomerStatusIncoming() customer.ID = %v, want %v", cu.ID, tt.expectCustomer.ID)
				}
			}
		})
	}
}
