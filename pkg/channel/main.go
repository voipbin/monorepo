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
	Tech       Tech

	// source/destination
	SourceName        string
	SourceNumber      string
	DestinationName   string
	DestinationNumber string

	State  ari.ChannelState
	Data   map[string]interface{}
	Stasis string

	DialResult  string
	HangupCause ari.ChannelCause

	TMCreate string
	TMUpdate string

	TMAnswer  string
	TMRinging string
	TMEnd     string
}

// Tech represent channel's technology
type Tech string

// List of Tech types
const (
	TechNone  Tech = ""
	TechLocal Tech = "local"
	TechPJSIP Tech = "pjsip"
	TechSIP   Tech = "sip"
	TechSnoop Tech = "snoop"
)

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
func getTech(name string) Tech {
	res := strings.Split(name, "/")
	if len(res) < 1 {
		return TechNone
	}

	tmp := strings.ToLower(res[0])
	switch tmp {
	case "pjsip":
		return TechPJSIP
	case "snoop":
		return TechSnoop
	case "local":
		return TechLocal
	case "sip":
		return TechSIP
	default:
		return TechNone
	}
}
