package utilhandler

import (
	"strings"
	"time"
)

// GetCurTime return current utc time string
func (h *utilHandler) GetCurTime() string {
	return GetCurTime()
}

// GetCurTimeAdd return return current utc time + duration string
func (h *utilHandler) GetCurTimeAdd(duration time.Duration) string {
	return GetCurTimeAdd(duration)
}

// GetCurTimeRFC3339 return current utc time string in a RFC3339 format
func (h *utilHandler) GetCurTimeRFC3339() string {
	return GetCurTimeRFC3339()
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// GetCurTimeAdd return current utc time + duration string
func GetCurTimeAdd(duration time.Duration) string {
	now := time.Now().Add(duration).UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// GetCurTimeRFC3339 return current utc time string in a RFC3339 format
func GetCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
