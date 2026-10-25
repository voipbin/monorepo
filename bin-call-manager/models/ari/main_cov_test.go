package ari

import (
	"testing"
	"time"
)

func Test_TimestampToTime_cov(t *testing.T) {
	tests := []struct {
		name   string
		input  Timestamp
		expect *time.Time
	}{
		{"empty", "", nil},
		{"millis layout", "2023-01-02T15:04:05.123", ptrTime(time.Date(2023, 1, 2, 15, 4, 5, 123000000, time.UTC))},
		{"seconds layout", "2023-01-02T15:04:05", ptrTime(time.Date(2023, 1, 2, 15, 4, 5, 0, time.UTC))},
		{"rfc3339", "2023-01-02T15:04:05Z", ptrTime(time.Date(2023, 1, 2, 15, 4, 5, 0, time.UTC))},
		{"rfc3339 nano", "2023-01-02T15:04:05.123456789Z", ptrTime(time.Date(2023, 1, 2, 15, 4, 5, 123456789, time.UTC))},
		{"invalid", "not-a-time", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input.ToTime()
			if tt.expect == nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}
			if res == nil || !res.Equal(*tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
