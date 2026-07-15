package emailhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Test_validateCustomerEmailRate exercises the per-customer outbound email
// rate limit gate: under-cap allows, at/over either window's cap fails
// closed, and a Redis error on either window also fails closed. VOIP-1259.
func Test_validateCustomerEmailRate(t *testing.T) {
	config.SetEmailOutboundRateLimitForTest(100, 1000)

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
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001"),
			minuteCount: 1,
			hourCount:   1,
			expectValid: true,
		},
		{
			name:        "exactly at minute cap - allowed",
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000002"),
			minuteCount: 100,
			hourCount:   100,
			expectValid: true,
		},
		{
			name:        "minute cap exceeded - rejected",
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000003"),
			minuteCount: 101,
			hourCount:   101,
			expectValid: false,
		},
		{
			name:        "exactly at hour cap - allowed",
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000004"),
			minuteCount: 1,
			hourCount:   1000,
			expectValid: true,
		},
		{
			name:        "hour cap exceeded - rejected",
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000005"),
			minuteCount: 1,
			hourCount:   1001,
			expectValid: false,
		},
		{
			name:           "redis error on minute counter - fail closed",
			customerID:     uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000006"),
			minuteErr:      fmt.Errorf("redis connection refused"),
			skipHourExpect: true,
			expectValid:    false,
		},
		{
			name:        "redis error on hour counter - fail closed",
			customerID:  uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000007"),
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

			h := &emailHandler{
				cache: mockCache,
			}

			ctx := context.Background()

			minuteKey := fmt.Sprintf("email-manager:ratelimit:email:%s:minute", tt.customerID)
			hourKey := fmt.Sprintf("email-manager:ratelimit:email:%s:hour", tt.customerID)

			mockCache.EXPECT().RateLimitIncrement(ctx, minuteKey, gomock.Any()).Return(tt.minuteCount, tt.minuteErr)
			if !tt.skipHourExpect {
				mockCache.EXPECT().RateLimitIncrement(ctx, hourKey, gomock.Any()).Return(tt.hourCount, tt.hourErr)
			}

			valid := h.validateCustomerEmailRate(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("validateCustomerEmailRate() valid = %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

// Test_validateCustomerEmailRate_IndependentCustomers confirms that two
// different customers do not share the same Redis rate-limit counters — each
// customer's key is scoped by customer_id. VOIP-1259.
func Test_validateCustomerEmailRate_IndependentCustomers(t *testing.T) {
	config.SetEmailOutboundRateLimitForTest(100, 1000)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &emailHandler{
		cache: mockCache,
	}

	ctx := context.Background()

	customerA := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000001")
	customerB := uuid.FromStringOrNil("e2000000-0000-0000-0000-000000000002")

	// customer A is already at the minute cap (101 > 100) — must be rejected.
	// Note: validateCustomerEmailRate unconditionally increments BOTH the
	// minute and hour counters before checking either cap (see design doc
	// §6-A: no early return, both RateLimitIncrement calls always happen),
	// so customer A's hour counter is also incremented even though the
	// minute check alone is what causes the rejection.
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("email-manager:ratelimit:email:%s:minute", customerA), gomock.Any()).Return(int64(101), nil)
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("email-manager:ratelimit:email:%s:hour", customerA), gomock.Any()).Return(int64(1), nil)

	// customer B is fresh (count 1) on both windows — must be allowed,
	// proving customer A's exhausted counter did not leak into customer B's
	// key.
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("email-manager:ratelimit:email:%s:minute", customerB), gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().RateLimitIncrement(ctx, fmt.Sprintf("email-manager:ratelimit:email:%s:hour", customerB), gomock.Any()).Return(int64(1), nil)

	if valid := h.validateCustomerEmailRate(ctx, customerA); valid {
		t.Errorf("validateCustomerEmailRate() customer A valid = %v, want false", valid)
	}
	if valid := h.validateCustomerEmailRate(ctx, customerB); !valid {
		t.Errorf("validateCustomerEmailRate() customer B valid = %v, want true", valid)
	}
}
