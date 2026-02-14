package utilhandler

import (
	"testing"
	"time"
)

func TestTimeGetCurTimeRFC3339(t *testing.T) {
	result := TimeGetCurTimeRFC3339()

	if len(result) == 0 {
		t.Error("TimeGetCurTimeRFC3339() returned empty string")
	}

	// Try parsing as RFC3339
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("TimeGetCurTimeRFC3339() = %s, not valid RFC3339: %v", result, err)
	}
}

func TestTimeParseWithError_AllFormats(t *testing.T) {
	tests := []struct {
		name       string
		timeString string
		wantErr    bool
	}{
		{
			name:       "ISO 8601 with Z",
			timeString: "2024-01-15T10:30:45.123456Z",
			wantErr:    false,
		},
		{
			name:       "ISO 8601 without Z",
			timeString: "2024-01-15T10:30:45.123456",
			wantErr:    false,
		},
		{
			name:       "Legacy format",
			timeString: "2024-01-15 10:30:45.123456",
			wantErr:    false,
		},
		{
			name:       "SQLite format with timezone",
			timeString: "2024-01-15 10:30:45.123456789-07:00",
			wantErr:    false,
		},
		{
			name:       "SQLite format with milliseconds",
			timeString: "2024-01-15 10:30:45.123-07:00",
			wantErr:    false,
		},
		{
			name:       "invalid format",
			timeString: "not-a-time",
			wantErr:    true,
		},
		{
			name:       "empty string",
			timeString: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TimeParseWithError(tt.timeString)

			if (err != nil) != tt.wantErr {
				t.Errorf("TimeParseWithError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.IsZero() {
					t.Error("TimeParseWithError() returned zero time for valid input")
				}
			}
		})
	}
}
