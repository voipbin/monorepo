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

// TimeParse parses the given time string
func (h *utilHandler) TimeParse(timeString string) time.Time {
	return TimeParse(timeString)
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

// TimeParse parse time string to the time.Time
func TimeParse(timeString string) time.Time {

	layout := "2006-01-02 15:04:05.000000"
	res, err := time.Parse(layout, timeString)
	if err != nil {
		return time.Time{}
	}

	return res
}
