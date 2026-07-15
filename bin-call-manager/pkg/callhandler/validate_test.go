package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/internal/config"
	"monorepo/bin-call-manager/pkg/cachehandler"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
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

func Test_validateOutgoingCallPermission(t *testing.T) {
	tests := []struct {
		name string

		customer    *cucustomer.Customer
		destination commonaddress.Address

		expectErr bool
	}{
		{
			name:     "nil customer - rejected",
			customer: nil,
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: true,
		},
		{
			name: "active verified customer with PSTN destination - allowed",
			customer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: false,
		},
		{
			name: "active verified customer with SIP destination - allowed",
			customer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "sip:user@example.com",
			},
			expectErr: false,
		},
		{
			name: "inactive customer - rejected",
			customer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000003"),
				Status:                     cucustomer.StatusFrozen,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: true,
		},
		{
			name: "active unverified customer with PSTN destination - rejected",
			customer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000004"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: true,
		},
		{
			name: "active unverified customer with SIP destination - allowed",
			customer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000005"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "sip:user@example.com",
			},
			expectErr: false,
		},
		{
			name: "internal customer ID CallManager with unverified PSTN - allowed",
			customer: &cucustomer.Customer{
				ID:                         cucustomer.IDCallManager,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: false,
		},
		{
			name: "internal customer ID AIManager with unverified PSTN - allowed",
			customer: &cucustomer.Customer{
				ID:                         cucustomer.IDAIManager,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: false,
		},
		{
			name: "internal customer ID System with unverified PSTN - allowed",
			customer: &cucustomer.Customer{
				ID:                         cucustomer.IDSystem,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: false,
		},
		{
			name: "internal customer ID BasicRoute with unverified PSTN - allowed",
			customer: &cucustomer.Customer{
				ID:                         cucustomer.IDBasicRoute,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551234567",
			},
			expectErr: false,
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

			err := h.validateOutgoingCallPermission(ctx, tt.customer, tt.destination)
			if tt.expectErr && err == nil {
				t.Errorf("validateOutgoingCallPermission() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateOutgoingCallPermission() expected nil, got %v", err)
			}
		})
	}
}

func Test_ValidateDestination(t *testing.T) {
	tests := []struct {
		name        string
		customerID  uuid.UUID
		config      *outboundconfig.OutboundConfig
		destination commonaddress.Address
		expectValid bool
	}{
		{
			name:       "non-tel destination (TypeSIP) - bypass - allowed",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
			config:     nil,
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "sip:user@example.com",
			},
			expectValid: true,
		},
		{
			name:       "internal customer IDCallManager + tel - bypass - allowed",
			customerID: cucustomer.IDCallManager,
			config:     nil,
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+12025550100",
			},
			expectValid: true,
		},
		{
			name:       "nil config + tel - deny",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000002"),
			config:     nil,
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+12025550100",
			},
			expectValid: false,
		},
		{
			name:       "empty whitelist config + tel - deny",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000003"),
			config:     &outboundconfig.OutboundConfig{DestinationWhitelist: []string{}},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+12025550100",
			},
			expectValid: false,
		},
		{
			name:       "config with [us] + US number - allowed",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000004"),
			config:     &outboundconfig.OutboundConfig{DestinationWhitelist: []string{"us"}},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+12025550100",
			},
			expectValid: true,
		},
		{
			name:       "config with [us] + UK number - blocked",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000005"),
			config:     &outboundconfig.OutboundConfig{DestinationWhitelist: []string{"us"}},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+442071234567",
			},
			expectValid: false,
		},
		{
			name:       "config with [us] + unparseable number - fail-closed - blocked",
			customerID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000006"),
			config:     &outboundconfig.OutboundConfig{DestinationWhitelist: []string{"us"}},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "notanumber",
			},
			expectValid: false,
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

			valid := h.ValidateDestination(ctx, tt.customerID, tt.config, tt.destination)
			if valid != tt.expectValid {
				t.Errorf("ValidateDestination() valid = %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

// Test_ValidateCustomerOutboundCallRate exercises the per-customer outbound call
// rate limit gate: under-cap allows, at/over either window's cap fails closed,
// and a Redis error on either window also fails closed. VOIP-1259.
func Test_ValidateCustomerOutboundCallRate(t *testing.T) {
	config.SetCallOutboundRateLimitForTest(20, 200)

	tests := []struct {
		name string

		customerID uuid.UUID

		minuteCount int64
		minuteErr   error
		hourCount   int64
		hourErr     error

		// skipHourExpect is set for cases where the minute window already fails
		// closed before RateLimitIncrement is invoked for the hour key.
		skipHourExpect bool

		expectValid bool
	}{
		{
			name:        "under both caps - allowed",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000001"),
			minuteCount: 1,
			hourCount:   1,
			expectValid: true,
		},
		{
			name:        "exactly at minute cap - allowed",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000002"),
			minuteCount: 20,
			hourCount:   20,
			expectValid: true,
		},
		{
			name:        "minute cap exceeded - rejected",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000003"),
			minuteCount: 21,
			hourCount:   21,
			expectValid: false,
		},
		{
			name:        "exactly at hour cap - allowed",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000004"),
			minuteCount: 1,
			hourCount:   200,
			expectValid: true,
		},
		{
			name:        "hour cap exceeded - rejected",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000005"),
			minuteCount: 1,
			hourCount:   201,
			expectValid: false,
		},
		{
			name:           "redis error on minute counter - fail closed",
			customerID:     uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000006"),
			minuteErr:      fmt.Errorf("redis connection refused"),
			skipHourExpect: true,
			expectValid:    false,
		},
		{
			name:        "redis error on hour counter - fail closed",
			customerID:  uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000007"),
			minuteCount: 1,
			hourErr:     fmt.Errorf("redis connection refused"),
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &callHandler{
				cache: mockCache,
			}

			ctx := context.Background()

			minuteKey := fmt.Sprintf("call-manager:ratelimit:call:%s:minute", tt.customerID)
			hourKey := fmt.Sprintf("call-manager:ratelimit:call:%s:hour", tt.customerID)

			mockCache.EXPECT().RateLimitIncrement(ctx, minuteKey, gomock.Any()).Return(tt.minuteCount, tt.minuteErr)
			if !tt.skipHourExpect {
				mockCache.EXPECT().RateLimitIncrement(ctx, hourKey, gomock.Any()).Return(tt.hourCount, tt.hourErr)
			}

			valid := h.ValidateCustomerOutboundCallRate(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("ValidateCustomerOutboundCallRate() valid = %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

// Test_ValidateCustomerOutboundCallRate_IndependentCustomers confirms that two
// different customers do not share the same Redis rate-limit counters — each
// customer's key is scoped by customer_id, mirroring
// TestRateLimit_DifferentIPsIndependent in
// bin-api-manager/lib/middleware/ratelimit_test.go. VOIP-1259.
func Test_ValidateCustomerOutboundCallRate_IndependentCustomers(t *testing.T) {
	config.SetCallOutboundRateLimitForTest(20, 200)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &callHandler{
		cache: mockCache,
	}

	ctx := context.Background()

	customerA := uuid.FromStringOrNil("f0000000-0000-0000-0000-000000000001")
	customerB := uuid.FromStringOrNil("f0000000-0000-0000-0000-000000000002")

	// customer A is already at the minute cap (21 > 20) — must be rejected.
	// Note: ValidateCustomerOutboundCallRate unconditionally increments BOTH
	// the minute and hour counters before checking either cap (see §6-A: no
	// early return, both RateLimitIncrement calls always happen), so customer
	// A's hour counter is also incremented even though the minute check alone
	// is what causes the rejection.
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("call-manager:ratelimit:call:%s:minute", customerA), gomock.Any()).Return(int64(21), nil)
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("call-manager:ratelimit:call:%s:hour", customerA), gomock.Any()).Return(int64(1), nil)

	// customer B is fresh (count 1) on both windows — must be allowed, proving
	// customer A's exhausted counter did not leak into customer B's key.
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("call-manager:ratelimit:call:%s:minute", customerB), gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("call-manager:ratelimit:call:%s:hour", customerB), gomock.Any()).Return(int64(1), nil)

	if valid := h.ValidateCustomerOutboundCallRate(ctx, customerA); valid {
		t.Errorf("ValidateCustomerOutboundCallRate() customer A valid = %v, want false", valid)
	}
	if valid := h.ValidateCustomerOutboundCallRate(ctx, customerB); !valid {
		t.Errorf("ValidateCustomerOutboundCallRate() customer B valid = %v, want true", valid)
	}
}
