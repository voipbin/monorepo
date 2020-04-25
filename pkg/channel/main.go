package channel

import (
	"strings"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

// Channel struct represent asterisk's channel information
type Channel struct {
	// identity
	ID         string
	AsteriskID string
	Name       string
	Tech       string

	// source/destination
	SourceName        string
	SourceNumber      string
	DestinationName   string
	DestinationNumber string

	State ari.ChannelState
	Data  map[string]interface{}

	DialResult  string
	HangupCause ari.ChannelCause

	TMCreate string
	TMUpdate string

	TMAnswer  string
	TMRinging string
	TMEnd     string
}

// NewChannelByChannelCreated creates Channel based on ARI ChannelCreated event
func NewChannelByChannelCreated(e *ari.ChannelCreated) *Channel {
	tech := getTech(e.Channel.Name)
	c := &Channel{
		AsteriskID: e.AsteriskID,
		ID:         e.Channel.ID,
		Name:       e.Channel.Name,
		Tech:       tech,

		SourceName:        e.Channel.Caller.Name,
		SourceNumber:      e.Channel.Caller.Number,
		DestinationNumber: e.Channel.Dialplan.Exten,

		State: e.Channel.State,
		Data:  make(map[string]interface{}, 1),

		TMCreate: string(e.Timestamp),
	}

	return c
}

// getTech returns tech from channel name
func getTech(name string) string {
	res := strings.Split(name, "/")
	if len(res) < 1 {
		return ""
	}

	return strings.ToLower(res[0])
}
