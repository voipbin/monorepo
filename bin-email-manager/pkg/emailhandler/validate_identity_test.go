package emailhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_validateCustomerIdentityVerified(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		// when responseCustomer and responseErr are both nil, the mock returns (nil, nil).
		// when expectCustomerGet is false, the RPC mock is NOT set (bypass path).
		expectCustomerGet bool
		responseCustomer  *cucustomer.Customer
		responseErr       error

		expectRes bool
	}{
		{
			name: "verified customer is allowed",

			customerID:        uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			expectCustomerGet: true,
			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},

			expectRes: true,
		},
		{
			name: "unverified customer is rejected",

			customerID:        uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			expectCustomerGet: true,
			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusPending,
			},

			expectRes: false,
		},
		{
			name: "internal system customer bypasses without RPC",

			customerID:        cucustomer.IDSystem,
			expectCustomerGet: false,

			expectRes: true,
		},
		{
			name: "fetch failure fails open",

			customerID:        uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			expectCustomerGet: true,
			responseErr:       fmt.Errorf("customer-manager unavailable"),

			expectRes: true,
		},
		{
			name: "nil customer with nil error fails open without panic",

			customerID:        uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			expectCustomerGet: true,
			responseCustomer:  nil,
			responseErr:       nil,

			expectRes: true,
		},
		{
			name: "suspended but verified customer is allowed (identity-only gating)",

			customerID:        uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			expectCustomerGet: true,
			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				Status:                     cucustomer.StatusFrozen,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},

			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
				cache:         mockCache,
			}
			ctx := context.Background()

			if tt.expectCustomerGet {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, tt.responseErr)
			}

			res := h.validateCustomerIdentityVerified(ctx, tt.customerID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_Create_unverified asserts that an unverified customer's email send is
// rejected before the balance check (no BillingV1AccountIsValidBalanceByCustomerID
// call, no email persisted).
func Test_Create_unverified(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &emailHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
		cache:         mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	activeflowID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")
	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeEmail,
			Target: "test@voipbin.net",
		},
	}

	// only the customer fetch is expected; balance check and DB create must NOT be called.
	mockReq.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:                         customerID,
		Status:                     cucustomer.StatusActive,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusRejected,
	}, nil)

	res, err := h.Create(ctx, customerID, activeflowID, destinations, "subject", "content", []email.Attachment{})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_Create_invalidDestination asserts that destination validation runs BEFORE
// the identity-verification gate: an invalid destination is rejected without any
// CustomerV1CustomerGet call.
func Test_Create_invalidDestination(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &emailHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
		cache:         mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	activeflowID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeTel, // not an email address -> invalid
			Target: "+821****0000",
		},
	}

	// no CustomerV1CustomerGet mock is set: destination validation must reject first.
	res, err := h.Create(ctx, customerID, activeflowID, destinations, "subject", "content", []email.Attachment{})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}
