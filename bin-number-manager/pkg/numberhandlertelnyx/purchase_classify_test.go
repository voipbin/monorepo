package numberhandlertelnyx

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/requestexternal"
	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
)

func Test_NumberPurchase_classification(t *testing.T) {
	num := "+155****0001"

	tests := []struct {
		name string

		responseOrder    *telnyx.OrderNumber
		responseOrderErr error

		expectLookup bool
		lookupErr    error

		expectRejected    bool
		expectUnconfirmed bool
	}{
		{name: "400 is a rejection", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusBadRequest}, expectRejected: true},
		{name: "401 is a rejection", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusUnauthorized}, expectRejected: true},
		{name: "403 is a rejection", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusForbidden}, expectRejected: true},
		{name: "404 is a rejection", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusNotFound}, expectRejected: true},
		{name: "422 is a rejection", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusUnprocessableEntity}, expectRejected: true},
		{name: "408 is unconfirmed", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusRequestTimeout}, expectUnconfirmed: true},
		{name: "409 is unconfirmed", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusConflict}, expectUnconfirmed: true},
		{name: "429 is unconfirmed", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusTooManyRequests}, expectUnconfirmed: true},
		{name: "500 is unconfirmed", responseOrderErr: &requestexternal.HTTPStatusError{StatusCode: http.StatusInternalServerError}, expectUnconfirmed: true},
		{name: "a network error is unconfirmed", responseOrderErr: errors.New("connection reset"), expectUnconfirmed: true},
		{name: "a wrapped status error is still classified", responseOrderErr: fmt.Errorf("outer: %w", &requestexternal.HTTPStatusError{StatusCode: http.StatusNotFound}), expectRejected: true},
		{name: "a failed order status is a rejection", responseOrder: &telnyx.OrderNumber{Status: "failure"}, expectRejected: true},
		{
			name:              "a pending order whose lookup fails is unconfirmed",
			responseOrder:     &telnyx.OrderNumber{Status: "pending"},
			expectLookup:      true,
			lookupErr:         errors.New("lookup failed"),
			expectUnconfirmed: true,
		},
		{
			name:              "an unknown order status whose lookup fails is unconfirmed",
			responseOrder:     &telnyx.OrderNumber{Status: "something-new"},
			expectLookup:      true,
			lookupErr:         errors.New("lookup failed"),
			expectUnconfirmed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			mockExternal.EXPECT().TelnyxNumberOrdersPost("test-token", []string{num}, "conn-1", "prof-1").Return(tt.responseOrder, tt.responseOrderErr)
			if tt.expectLookup {
				mockExternal.EXPECT().TelnyxPhoneNumbersGetByNumber("test-token", num).Return(nil, tt.lookupErr)
			}

			res, err := h.NumberPurchase(num)
			if res != nil || err == nil {
				t.Fatalf("Wrong match. expect: error, got: %v, %v", res, err)
			}
			if errors.Is(err, provider.ErrPurchaseRejected) != tt.expectRejected {
				t.Errorf("Wrong rejected classification. expect: %v, got error: %v", tt.expectRejected, err)
			}
			if errors.Is(err, provider.ErrPurchaseUnconfirmed) != tt.expectUnconfirmed {
				t.Errorf("Wrong unconfirmed classification. expect: %v, got error: %v", tt.expectUnconfirmed, err)
			}
		})
	}
}
