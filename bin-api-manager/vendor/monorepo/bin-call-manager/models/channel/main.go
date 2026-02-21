package channel

import (
	"strings"
	"time"

	"monorepo/bin-call-manager/models/ari"
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

	State      ari.ChannelState          `json:"state"`
	Data       map[string]interface{}    `json:"data"`
	StasisName string                    `json:"stasis_name"` // stasis name
	StasisData map[StasisDataType]string `json:"stasis_data"` // stasis data
	BridgeID   string                    `json:"bridge_id"`
	PlaybackID string                    `json:"playback_id"` // playback id

	DialResult  string           `json:"dial_result"`
	HangupCause ari.ChannelCause `json:"hangup_cause"`

	Direction     Direction     `json:"direction"`
	MuteDirection MuteDirection `json:"mute_direction"`

	TMAnswer  *time.Time `json:"tm_answer"`
	TMRinging *time.Time `json:"tm_ringing"`
	TMEnd     *time.Time `json:"tm_end"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// Tech represent channel's technology
type Tech string

// List of Tech types
const (
	TechNone        Tech = ""
	TechAudioSocket Tech = "audiosocket"
	TechLocal       Tech = "local"
	TechPJSIP       Tech = "pjsip"
	TechSIP         Tech = "sip"
	TechSnoop       Tech = "snoop"
	TechUnicatRTP   Tech = "unicastrtp" // external media
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

// Context defines channel's context
type Context string

// List of context
const (
	// call
	ContextCallIncoming  Context = "call-in"            // context for the incoming channel
	ContextCallOutgoing  Context = "call-out"           // context for the outgoing channel
	ContextCallRecovery  Context = "call-recovery"      // context for the channel which created for recovery
	ContextRecording     Context = "call-record"        // context for the channel which created only for recording
	ContextCallService   Context = "call-svc"           // context for the channel where it came back to stasis from the other asterisk application
	ContextJoinCall      Context = "call-join"          // context for the channel for conference joining
	ContextExternalMedia Context = "call-externalmedia" // context for the external media channel. this channel will get the media from the external
	ContextExternalSnoop Context = "call-externalsnoop" // context for the external snoop channel
	ContextApplication   Context = "call-application"   // context for dialplan application execution

	// conf
	ContextConfIncoming Context = "conf-in"  // context for the incoming channel to the conference-asterisk
	ContextConfOutgoing Context = "conf-out" // context for the outgoing channel from the conference-asterisk
)

// SIPTransport represent channel's sip transport type.
type SIPTransport string

// List of SIPTransport types
const (
	SIPTransportNone SIPTransport = ""
	SIPTransportUDP  SIPTransport = "udp"
	SIPTransportTCP  SIPTransport = "tcp"
	SIPTransportTLS  SIPTransport = "tls"
	SIPTransportWS   SIPTransport = "ws"
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

// MuteDirection represents possible values for channel mute
type MuteDirection string

// List of mute direction types
const (
	MuteDirectionNone MuteDirection = ""     // none
	MuteDirectionBoth MuteDirection = "both" // mute the channel in/out both.
	MuteDirectionOut  MuteDirection = "out"  //
	MuteDirectionIn   MuteDirection = "in"   // mute the channel incoming
)

// ContextType represent channel's context type.
// it points which handler needs to handle the channel.
// normally, it designed for which asterisk type generated the channel(ex asterisk-call, asterisk-conference).
// but it can be used for type's for context as well.
type ContextType string

// List of ContextType types.
const (
	ContextTypeConference ContextType = "conf" // conference type context. callhandler needs to handle this channel.
	ContextTypeCall       ContextType = "call" // call type context. conferencehandler needs to handle this channel.
)

// StasisDataType represents channel's stasis data types
type StasisDataType string

// List of StasisDataType types
const (
	// voipbin dependent types
	StasisDataTypeContextType     StasisDataType = "context_type"      // type: channel's contexr type
	StasisDataTypeContext         StasisDataType = "context"           // context: channel's context
	StasisDataTypeContextFrom     StasisDataType = "context_from"      // represents based context. used for service return stasis arg
	StasisDataTypeDomain          StasisDataType = "domain"            // requested domain name.
	StasisDataTypeSource          StasisDataType = "source"            // request source ip
	StasisDataTypeDirection       StasisDataType = "direction"         // channel's direction. incoming for this case.
	StasisDataTypeCallID          StasisDataType = "call_id"           // voipbin call id
	StasisDataTypeConfbridgeID    StasisDataType = "confbridge_id"     // voipbin confbridge id
	StasisDataTypeExternalMediaID StasisDataType = "external_media_id" // voipbin external media id
	StasisDataTypeTransport       StasisDataType = "transport"         // channel's transport(SIP transport)
	StasisDataTypeApplicationName StasisDataType = "application_name"  // application name
	StasisDataTypeBridgeID        StasisDataType = "bridge_id"         // bridge's id
	StasisDataTypeReferenceType   StasisDataType = "reference_type"    // given channel's reference type
	StasisDataTypeReferenceID     StasisDataType = "reference_id"      // given channel's reference id

	// SIP dependent types
	StasisDataTypeSIPCallID  StasisDataType = "sip_call_id" // SIP Call-ID
	StasisDataTypeSIPPAI     StasisDataType = "sip_pai"     // SIP P-Asserted-Identity
	StasisDataTypeSIPPrivacy StasisDataType = "sip_privacy" // SIP Privacy

	// recording
	StasisDataTypeRecordingID           StasisDataType = "recording_id"
	StasisDataTypeRecordingName         StasisDataType = "recording_name"
	StasisDataTypeRecordingDirection    StasisDataType = "recording_direction"
	StasisDataTypeRecordingFormat       StasisDataType = "recording_format"
	StasisDataTypeRecordingEndOfSilence StasisDataType = "recording_end_of_silence"
	StasisDataTypeRecordingEndOfKey     StasisDataType = "recording_end_of_key"
	StasisDataTypeRecordingDuration     StasisDataType = "recording_duration"

	// service - amd
	StasisDataTypeServiceAMDStatus StasisDataType = "amd_status" // amd result status
	StasisDataTypeServiceAMDCause  StasisDataType = "amd_cause"  // amd result cause
)

// NewChannelByChannelCreated creates Channel based on ARI ChannelCreated event
func NewChannelByChannelCreated(e *ari.ChannelCreated) *Channel {
	c := NewChannelByARIChannel(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = e.Timestamp.ToTime()

	return c
}

// NewChannelByStasisStart creats a Channel based on ARI StasisStart event
func NewChannelByStasisStart(e *ari.StasisStart) *Channel {
	c := NewChannelByARIChannel(&e.Channel)
	c.AsteriskID = e.AsteriskID
	c.TMCreate = e.Timestamp.ToTime()

	// get stasis name and stasis data
	stasisData := map[StasisDataType]string{}
	for k, v := range e.Args {
		stasisData[StasisDataType(k)] = v
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
		StasisData: map[StasisDataType]string{},
	}

	for k, i := range e.ChannelVars {
		c.Data[k] = i
	}

	return c
}

// GetTech returns tech from channel name
func GetTech(name string) Tech {
	tmps := strings.Split(name, "/")
	if len(tmps) < 1 {
		return TechNone
	}

	mapTechs := map[string]Tech{
		string(TechAudioSocket): TechAudioSocket,
		string(TechLocal):       TechLocal,
		string(TechPJSIP):       TechPJSIP,
		string(TechSIP):         TechSIP,
		string(TechSnoop):       TechSnoop,
		string(TechUnicatRTP):   TechUnicatRTP,
	}

	tmp := strings.ToLower(tmps[0])
	res, ok := mapTechs[tmp]
	if !ok {
		return TechNone
	}

	return res
}
