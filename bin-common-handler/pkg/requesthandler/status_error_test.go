package requesthandler

import (
	"errors"
	"net/http"
	"testing"
)

// Test_getResponseStatusCodeError_Sentinels fills gaps in the existing
// Test_converError_error table (400, 500, 499, 599 only): the sentinel
// mapping for 404 and 409, and the < 300 boundary (200, 299 are nil, 300
// falls through to ErrUnknown like any other unmapped code).
func Test_getResponseStatusCodeError_Sentinels(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int

		expectSentinel error
	}{
		{
			name:           "200 is below the error threshold",
			statusCode:     http.StatusOK,
			expectSentinel: nil,
		},
		{
			name:           "299 is below the error threshold",
			statusCode:     299,
			expectSentinel: nil,
		},
		{
			name:           "300 is unmapped",
			statusCode:     300,
			expectSentinel: ErrUnknown,
		},
		{
			name:           "400 maps to ErrBadRequest",
			statusCode:     http.StatusBadRequest,
			expectSentinel: ErrBadRequest,
		},
		{
			name:           "404 maps to ErrNotFound",
			statusCode:     http.StatusNotFound,
			expectSentinel: ErrNotFound,
		},
		{
			name:           "409 maps to ErrConflict",
			statusCode:     http.StatusConflict,
			expectSentinel: ErrConflict,
		},
		{
			name:           "500 maps to ErrInternal",
			statusCode:     http.StatusInternalServerError,
			expectSentinel: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := getResponseStatusCodeError(tt.statusCode)

			if tt.expectSentinel == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", err)
				}
				return
			}

			if !errors.Is(err, tt.expectSentinel) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectSentinel, err)
			}
		})
	}
}
