package channel

import (
	"strings"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// Channel struct represent asterisk's channel information
type Channel struct {
	// identity
	ID         string `json:"id"`
	AsteriskID string `json:"asterisk_id"`
	Name       string `json:"name"`
	Type       Type   `json:"type"`
	Tech       Tech   `json:"tech"`

	// sip information
	SIPCallID    string       `json:"sip_call_id"`   // sip's call id
	SIPTransport SIPTransport `json:"sip_transport"` // sip's transport

	// source/destination
	SourceName        string `json:"source_name"`
	SourceNumber      string `json:"source_number"`
	DestinationName   string `json:"destination_name"`
	DestinationNumber string `json:"destination_number"`

	State      ari.ChannelState       `json:"state"`
	Data       map[string]interface{} `json:"data"`
	StasisName string                 `json:"stasis_name"` // stasis name
	StasisData map[string]string      `json:"stasis_data"` // stasis data
	BridgeID   string                 `json:"bridge_id"`
	PlaybackID string                 `json:"playback_id"` // playback id

	DialResult  string           `json:"dial_result"`
	HangupCause ari.ChannelCause `json:"hangup_cause"`

	Direction Direction `json:"direction"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMAnswer  string `json:"tm_answer"`
	TMRinging string `json:"tm_ringing"`
	TMEnd     string `json:"tm_end"`
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
	TypeNone        Type = ""            // the type has not defined yet.
	TypeCall        Type = "call"        // call channel
	TypeConfbridge  Type = "confbridge"  // confbridge channel
	TypeJoin        Type = "join"        // joining channel
	TypeExternal    Type = "external"    // channel for the external channel(snoop/media)
	TypeRecording   Type = "recording"   // channel for the recording
	TypeApplication Type = "application" // general purpose for do nothing
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

	// get stasis name and stasis data
	stasisData := map[string]string{}
	for k, v := range e.Args {
		stasisData[k] = v
	}
	c.StasisName = e.Application
	c.StasisData = stasisData

	return c
}

// NewChannelByARIChannel returns partial of channel struct
func NewChannelByARIChannel(e *ari.Channel) *Channel {
	tech := GetTech(e.Name)
	c := &Channel{
		ID:   e.ID,
		Name: e.Name,
		Tech: tech,

		SourceName:        e.Caller.Name,
		SourceNumber:      e.Caller.Number,
		DestinationNumber: e.Dialplan.Exten,

		State:      e.State,
		Data:       map[string]interface{}{},
		StasisData: map[string]string{},
	}

	for k, i := range e.ChannelVars {
		c.Data[k] = i
	}

	return c
}

// GetTech returns tech from channel name
func GetTech(name string) Tech {
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
