package utilhandler

import (
	"regexp"
	"testing"
	"time"
)

func Test_TimeGetCurTime_ISO8601Format(t *testing.T) {
	result := TimeGetCurTime()

	// Verify ISO 8601 format with microseconds: 2024-01-15T10:30:45.123456Z
	iso8601Regex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)
	if !iso8601Regex.MatchString(result) {
		t.Errorf("Expected ISO 8601 format, got: %s", result)
	}
}

func Test_TimeGetCurTimeAdd_ISO8601Format(t *testing.T) {
	result := TimeGetCurTimeAdd(time.Hour)

	// Verify ISO 8601 format with microseconds
	iso8601Regex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)
	if !iso8601Regex.MatchString(result) {
		t.Errorf("Expected ISO 8601 format, got: %s", result)
	}
}

func Test_TimeParse_ISO8601Format(t *testing.T) {
	tests := []struct {
		name       string
		timeString string
		expectYear int
		expectMon  time.Month
		expectDay  int
		expectHour int
		expectMin  int
		expectSec  int
	}{
		{
			name:       "ISO 8601 format",
			timeString: "2023-06-08T03:22:17.995001Z",
			expectYear: 2023,
			expectMon:  time.June,
			expectDay:  8,
			expectHour: 3,
			expectMin:  22,
			expectSec:  17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := TimeParse(tt.timeString)

			if res.Year() != tt.expectYear || res.Month() != tt.expectMon || res.Day() != tt.expectDay ||
				res.Hour() != tt.expectHour || res.Minute() != tt.expectMin || res.Second() != tt.expectSec {
				t.Errorf("Wrong match.\nexpect: %d-%02d-%02d %02d:%02d:%02d\ngot: %v",
					tt.expectYear, tt.expectMon, tt.expectDay, tt.expectHour, tt.expectMin, tt.expectSec, res)
			}
		})
	}
}

func Test_TimeParse_LegacyFormat(t *testing.T) {
	tests := []struct {
		name       string
		timeString string
		expectYear int
		expectMon  time.Month
		expectDay  int
		expectHour int
		expectMin  int
		expectSec  int
	}{
		{
			name:       "legacy format (backward compatibility)",
			timeString: "2023-06-08T03:22:17.995001Z",
			expectYear: 2023,
			expectMon:  time.June,
			expectDay:  8,
			expectHour: 3,
			expectMin:  22,
			expectSec:  17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := TimeParse(tt.timeString)

			if res.Year() != tt.expectYear || res.Month() != tt.expectMon || res.Day() != tt.expectDay ||
				res.Hour() != tt.expectHour || res.Minute() != tt.expectMin || res.Second() != tt.expectSec {
				t.Errorf("Wrong match.\nexpect: %d-%02d-%02d %02d:%02d:%02d\ngot: %v",
					tt.expectYear, tt.expectMon, tt.expectDay, tt.expectHour, tt.expectMin, tt.expectSec, res)
			}
		})
	}
}

func Test_TimeParseWithError(t *testing.T) {
	tests := []struct {
		name       string
		timeString string
		expectErr  bool
		expectYear int
		expectMon  time.Month
		expectDay  int
	}{
		{
			name:       "valid ISO 8601 time",
			timeString: "2023-06-08T03:22:17.995001Z",
			expectErr:  false,
			expectYear: 2023,
			expectMon:  time.June,
			expectDay:  8,
		},
		{
			name:       "valid legacy time (backward compatibility)",
			timeString: "2023-06-08T03:22:17.995001Z",
			expectErr:  false,
			expectYear: 2023,
			expectMon:  time.June,
			expectDay:  8,
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
				if res.Year() != tt.expectYear || res.Month() != tt.expectMon || res.Day() != tt.expectDay {
					t.Errorf("Wrong match.\nexpect: %d-%02d-%02d\ngot: %v",
						tt.expectYear, tt.expectMon, tt.expectDay, res)
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

func Test_ISO8601Layout_Constant(t *testing.T) {
	expected := "2006-01-02T15:04:05.000000Z"
	if ISO8601Layout != expected {
		t.Errorf("ISO8601Layout constant mismatch.\nexpect: %v\ngot: %v", expected, ISO8601Layout)
	}
}

func Test_LegacyLayout_Constant(t *testing.T) {
	expected := "2006-01-02 15:04:05.000000"
	if LegacyLayout != expected {
		t.Errorf("LegacyLayout constant mismatch.\nexpect: %v\ngot: %v", expected, LegacyLayout)
	}
}

func Test_TimeGetCurTime_RoundTrip(t *testing.T) {
	// Get current time as ISO 8601 string
	timeStr := TimeGetCurTime()

	// Parse it back
	parsed, err := TimeParseWithError(timeStr)
	if err != nil {
		t.Errorf("Failed to parse TimeGetCurTime output: %v", err)
	}

	// Verify it's a valid time (not zero)
	if parsed.IsZero() {
		t.Errorf("Parsed time is zero")
	}

	// Verify the time is recent (within last minute)
	now := time.Now().UTC()
	diff := now.Sub(parsed)
	if diff < 0 || diff > time.Minute {
		t.Errorf("Parsed time is not recent. Now: %v, Parsed: %v, Diff: %v", now, parsed, diff)
	}
}

// Tests for new *time.Time helper functions

func Test_TimeNow(t *testing.T) {
	// Get current time as pointer
	result := TimeNow()

	// Verify result is not nil
	if result == nil {
		t.Fatal("TimeNow() returned nil")
	}

	// Verify it's in UTC
	if result.Location() != time.UTC {
		t.Errorf("TimeNow() not in UTC. Location: %v", result.Location())
	}

	// Verify it's recent (within last second)
	now := time.Now().UTC()
	diff := now.Sub(*result)
	if diff < 0 || diff > time.Second {
		t.Errorf("TimeNow() is not recent. Now: %v, Result: %v, Diff: %v", now, *result, diff)
	}
}

func Test_TimeNowAdd(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"add 1 hour", time.Hour},
		{"add 30 minutes", 30 * time.Minute},
		{"subtract 1 hour", -time.Hour},
		{"add 0", 0},
		{"add 24 hours", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			result := TimeNowAdd(tt.duration)
			after := time.Now().UTC()

			// Verify result is not nil
			if result == nil {
				t.Fatal("TimeNowAdd() returned nil")
			}

			// Verify it's in UTC
			if result.Location() != time.UTC {
				t.Errorf("TimeNowAdd() not in UTC. Location: %v", result.Location())
			}

			// Verify the time is correct (between before+duration and after+duration)
			expectedMin := before.Add(tt.duration)
			expectedMax := after.Add(tt.duration)

			if result.Before(expectedMin) || result.After(expectedMax) {
				t.Errorf("TimeNowAdd(%v) out of range. Result: %v, Expected: [%v, %v]",
					tt.duration, *result, expectedMin, expectedMax)
			}
		})
	}
}

func Test_IsDeleted(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    *time.Time
		expected bool
	}{
		{
			name:     "nil means not deleted",
			input:    nil,
			expected: false,
		},
		{
			name:     "non-nil time means deleted",
			input:    &now,
			expected: true,
		},
		{
			name:     "zero time means deleted (pointer is not nil)",
			input:    func() *time.Time { t := time.Time{}; return &t }(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeleted(tt.input)
			if result != tt.expected {
				t.Errorf("IsDeleted(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_utilHandler_TimeNow(t *testing.T) {
	h := NewUtilHandler()

	result := h.TimeNow()

	if result == nil {
		t.Fatal("TimeNow() returned nil")
	}

	// Verify it's in UTC and recent
	now := time.Now().UTC()
	diff := now.Sub(*result)
	if diff < 0 || diff > time.Second {
		t.Errorf("TimeNow() is not recent. Now: %v, Result: %v, Diff: %v", now, *result, diff)
	}
}

func Test_utilHandler_TimeNowAdd(t *testing.T) {
	h := NewUtilHandler()

	duration := time.Hour
	before := time.Now().UTC()
	result := h.TimeNowAdd(duration)
	after := time.Now().UTC()

	if result == nil {
		t.Fatal("TimeNowAdd() returned nil")
	}

	expectedMin := before.Add(duration)
	expectedMax := after.Add(duration)

	if result.Before(expectedMin) || result.After(expectedMax) {
		t.Errorf("TimeNowAdd(%v) out of range. Result: %v, Expected: [%v, %v]",
			duration, *result, expectedMin, expectedMax)
	}
}

func Test_utilHandler_IsDeleted(t *testing.T) {
	h := NewUtilHandler()

	// Test nil - not deleted
	if h.IsDeleted(nil) {
		t.Error("IsDeleted(nil) should return false")
	}

	// Test non-nil - deleted
	now := time.Now().UTC()
	if !h.IsDeleted(&now) {
		t.Error("IsDeleted(&now) should return true")
	}
}
