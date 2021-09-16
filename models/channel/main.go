package channel

import (
	"strings"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// Channel struct represent asterisk's channel information
type Channel struct {
	// identity
	ID         string
	AsteriskID string
	Name       string
	Type       Type
	Tech       Tech

	// sip information
	SIPCallID    string       // sip's call id
	SIPTransport SIPTransport // sip's transport

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

	Direction Direction

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
	TechNone      Tech = ""
	TechLocal     Tech = "local"
	TechPJSIP     Tech = "pjsip"
	TechSIP       Tech = "sip"
	TechSnoop     Tech = "snoop"
	TechUnicatRTP Tech = "unicastrtp" // external media
)

// Type represent channel's type.
type Type string

// List of Context types
const (
	TypeNone      Type = ""          // the type has not defined yet.
	TypeCall      Type = "call"      // call channel
	TypeConf      Type = "conf"      // conference channel
	TypeJoin      Type = "join"      // joining channel
	TypeExternal  Type = "external"  // channel for the external channel(snoop/media)
	TypeRecording Type = "recording" // channel for the recording
)

// SIPTransport represent channel's sip transport type.
type SIPTransport string

// List of SIPTransport types
const (
	SIPTransportNone SIPTransport = ""
	SIPTransportUDP  SIPTransport = "udp"
	SIPTransportTCP  SIPTransport = "tcp"
	SIPTransportTLS  SIPTransport = "tls"
	SIPTransportWSS  SIPTransport = "wss"
)

// Direction represent channel's direction.
type Direction string

// List of Direction types
const (
	DirectionNone     Direction = ""
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
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

// ContextType represent channel's context type.
type ContextType string

// List of ContextType types.
const (
	ContextTypeConference ContextType = "conf"
	ContextTypeCall       ContextType = "call"
)

// NewChannelByChannelCreated creates Channel based on ARI ChannelCreated event
func NewChannelByChannelCreated(e *ari.ChannelCreated) *Channel {
	c := NewChannelByARIChannel(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = string(e.Timestamp)

	return c
}

// NewChannelByStasisStart creats a Channel based on ARI StasisStart event
func NewChannelByStasisStart(e *ari.StasisStart) *Channel {
	c := NewChannelByARIChannel(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = string(e.Timestamp)

	return c
}

// NewChannelByARIChannel returns partial of channel struct
func NewChannelByARIChannel(e *ari.Channel) *Channel {
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

	for k, i := range e.ChannelVars {
		c.Data[k] = i
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
	case "unicastrtp":
		return TechUnicatRTP
	default:
		return TechNone
	}
}
