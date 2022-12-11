package utilhandler

import (
	"strings"
	"time"
)

// GetCurTime return current utc time string
func (h *utilHandler) GetCurTime() string {
	return GetCurTime()
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

// GetCurTimeRFC3339 return current utc time string in a RFC3339 format
func GetCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
