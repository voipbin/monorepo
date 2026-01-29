package utilhandler

import (
	"strings"
	"time"
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

// TimeGetCurTime return current utc time string
func TimeGetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// TimeGetCurTimeAdd return current utc time + duration string
func TimeGetCurTimeAdd(duration time.Duration) string {
	now := time.Now().Add(duration).UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
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
func TimeParseWithError(timeString string) (time.Time, error) {
	layout := "2006-01-02 15:04:05.000000"
	return time.Parse(layout, timeString)
}
