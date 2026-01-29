package utilhandler

import (
	"testing"
	"time"
)

func Test_TimeConvert(t *testing.T) {

	type test struct {
		name string

		timeString string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			timeString: "2023-06-08 03:22:17.995001",
			expectRes:  "2023-06-08 03:22:17.995001 +0000 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := TimeParse(tt.timeString)

			if tt.expectRes != res.String() {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TimeParseWithError(t *testing.T) {
	tests := []struct {
		name        string
		timeString  string
		expectErr   bool
		expectTime  string
	}{
		{
			name:       "valid time",
			timeString: "2023-06-08 03:22:17.995001",
			expectErr:  false,
			expectTime: "2023-06-08 03:22:17.995001 +0000 UTC",
		},
		{
			name:       "invalid format",
			timeString: "not-a-time",
			expectErr:  true,
		},
		{
			name:       "empty string",
			timeString: "",
			expectErr:  true,
		},
		{
			name:       "wrong layout",
			timeString: "2023/06/08 03:22:17",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := TimeParseWithError(tt.timeString)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if !res.IsZero() {
					t.Errorf("Expected zero time on error, got: %v", res)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectTime != res.String() {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectTime, res)
				}
			}
		})
	}
}

func Test_TimeParseReturnsZeroOnError(t *testing.T) {
	// TimeParse should return zero time on invalid input without panic
	res := TimeParse("invalid-time-string")
	if !res.IsZero() {
		t.Errorf("Expected zero time for invalid input, got: %v", res)
	}
	if res != (time.Time{}) {
		t.Errorf("Expected time.Time{} for invalid input, got: %v", res)
	}
}
