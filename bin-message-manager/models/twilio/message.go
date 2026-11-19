package twilio

import (
	"strconv"
	"strings"
)

// MessageResponse is the part of the answer of the Twilio message create request that the service uses.
type MessageResponse struct {
	// SID is the id of the message (SM or MM followed by 32 hex digits).
	SID string `json:"sid"`

	// NumSegments is the segment count. Twilio sends it as a string and it can be "0" right after the
	// message is queued.
	NumSegments any `json:"num_segments"`

	// Status is the status of the message when it was created (normally queued).
	Status string `json:"status"`
}

// Segments returns the segment count, or 0 when Twilio gave none or a value that is not a number.
func (m *MessageResponse) Segments() int {
	switch v := m.NumSegments.(type) {
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 0 {
			return 0
		}
		return n
	case float64:
		if v < 0 {
			return 0
		}
		return int(v)
	default:
		return 0
	}
}
