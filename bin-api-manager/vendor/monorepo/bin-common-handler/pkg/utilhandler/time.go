package utilhandler

import (
	"fmt"
	"time"
)

const (
	// ISO8601Layout is the standard ISO 8601 format with microsecond precision
	ISO8601Layout = "2006-01-02T15:04:05.000000Z"
	// ISO8601LayoutNoZ is the ISO 8601 format without trailing Z (for parsing only)
	ISO8601LayoutNoZ = "2006-01-02T15:04:05.000000"
	// LegacyLayout is the old custom format for backward compatibility
	LegacyLayout = "2006-01-02 15:04:05.000000"
	// SQLiteLayout is the SQLite datetime format with timezone offset
	SQLiteLayout = "2006-01-02 15:04:05.999999999-07:00"
	// SQLiteLayoutMillis is the SQLite datetime format with milliseconds and timezone offset
	SQLiteLayoutMillis = "2006-01-02 15:04:05.999-07:00"
)

// TimeGetCurTime return current utc time string
func (h *utilHandler) TimeGetCurTime() string {
	return TimeGetCurTime()
}

// TimeGetCurTimeAdd return return current utc time + duration string
func (h *utilHandler) TimeGetCurTimeAdd(duration time.Duration) string {
	return TimeGetCurTimeAdd(duration)
}

// TimeGetCurTimeRFC3339 return current utc time string in a RFC3339 format
func (h *utilHandler) TimeGetCurTimeRFC3339() string {
	return TimeGetCurTimeRFC3339()
}

// TimeParse parses the given time string.
// Returns zero time on parse failure. Use TimeParseWithError for error details.
func (h *utilHandler) TimeParse(timeString string) time.Time {
	return TimeParse(timeString)
}

// TimeParseWithError parses the given time string and returns any parsing error.
func (h *utilHandler) TimeParseWithError(timeString string) (time.Time, error) {
	return TimeParseWithError(timeString)
}

// TimeGetCurTime return current utc time string in ISO 8601 format
func TimeGetCurTime() string {
	return time.Now().UTC().Format(ISO8601Layout)
}

// TimeGetCurTimeAdd return current utc time + duration string in ISO 8601 format
func TimeGetCurTimeAdd(duration time.Duration) string {
	return time.Now().Add(duration).UTC().Format(ISO8601Layout)
}

// TimeGetCurTimeRFC3339 return current utc time string in a RFC3339 format
func TimeGetCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// TimeParse parses time string to time.Time.
// Returns zero time on parse failure. Use TimeParseWithError for error details.
func TimeParse(timeString string) time.Time {
	res, _ := TimeParseWithError(timeString)
	return res
}

// TimeParseWithError parses time string to time.Time and returns any parsing error.
// This allows callers to detect and handle invalid time strings appropriately.
// Supports both ISO 8601 format and legacy custom format for backward compatibility.
func TimeParseWithError(timeString string) (time.Time, error) {
	// Try ISO 8601 format with Z suffix first (new format)
	if t, err := time.Parse(ISO8601Layout, timeString); err == nil {
		return t, nil
	}

	// Try ISO 8601 format without Z suffix
	if t, err := time.Parse(ISO8601LayoutNoZ, timeString); err == nil {
		return t, nil
	}

	// Try legacy format for backward compatibility
	if t, err := time.Parse(LegacyLayout, timeString); err == nil {
		return t, nil
	}

	// Try SQLite datetime format with timezone offset
	if t, err := time.Parse(SQLiteLayout, timeString); err == nil {
		return t, nil
	}

	// Try SQLite datetime format with milliseconds and timezone offset
	if t, err := time.Parse(SQLiteLayoutMillis, timeString); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeString)
}

// TimeNow returns a pointer to the current UTC time.
// Returns a *time.Time for use with nullable timestamp fields.
func (h *utilHandler) TimeNow() *time.Time {
	return TimeNow()
}

// TimeNowAdd returns a pointer to the current UTC time plus the given duration.
// Returns a *time.Time for use with nullable timestamp fields.
func (h *utilHandler) TimeNowAdd(d time.Duration) *time.Time {
	return TimeNowAdd(d)
}

// IsDeleted returns true if the timestamp pointer is not nil.
// Used for soft delete semantics: nil = not deleted, non-nil = deleted at that time.
func (h *utilHandler) IsDeleted(t *time.Time) bool {
	return IsDeleted(t)
}

// TimeNow returns a pointer to the current UTC time.
// Returns a *time.Time for use with nullable timestamp fields.
func TimeNow() *time.Time {
	t := time.Now().UTC()
	return &t
}

// TimeNowAdd returns a pointer to the current UTC time plus the given duration.
// Returns a *time.Time for use with nullable timestamp fields.
func TimeNowAdd(d time.Duration) *time.Time {
	t := time.Now().Add(d).UTC()
	return &t
}

// IsDeleted returns true if the timestamp pointer is not nil.
// Used for soft delete semantics: nil = not deleted, non-nil = deleted at that time.
func IsDeleted(t *time.Time) bool {
	return t != nil
}
