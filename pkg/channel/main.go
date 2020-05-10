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

	State    ari.ChannelState
	Data     map[string]interface{}
	Stasis   string
	BridgeID string

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

// SnoopDirection represents possible values for channel snoop
type SnoopDirection string

// List of ChannelSnoopType types
const (
	SnoopDirectionNone SnoopDirection = ""     // none
	SnoopDirectionBoth SnoopDirection = "both" // snoop the channel in/out both.
	SnoopDirectionOut  SnoopDirection = "out"  //
	SnoopDirectionIn   SnoopDirection = "in"   // snoop the channel incoming
)

// NewChannelByChannelCreated creates Channel based on ARI ChannelCreated event
func NewChannelByChannelCreated(e *ari.ChannelCreated) *Channel {
	c := newChannelByChannelStruct(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = string(e.Timestamp)

	return c
}

// NewChannelByStasisStart creats a Channel based on ARI StasisStart event
func NewChannelByStasisStart(e *ari.StasisStart) *Channel {
	c := newChannelByChannelStruct(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = string(e.Timestamp)

	return c
}

// newChannelByChannelStruct returns partial of channel struct
func newChannelByChannelStruct(e *ari.Channel) *Channel {
	tech := getTech(e.Name)
	c := &Channel{
		ID:   e.ID,
		Name: e.Name,
		Tech: tech,

		SourceName:        e.Caller.Name,
		SourceNumber:      e.Caller.Number,
		DestinationNumber: e.Dialplan.Exten,

		State: e.State,
		Data:  make(map[string]interface{}, 1),
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
