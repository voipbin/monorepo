package externalmedia

import (
	"github.com/gofrs/uuid"
)

// ExternalMedia defines external media detail info
type ExternalMedia struct {
	ID   uuid.UUID `json:"id"`
	Type Type      `json:"type"` // type of the external media

	AsteriskID string `json:"asterisk_id"`           // asterisk id
	ChannelID  string `json:"channel_id"`            // external media channel id
	BridgeID   string `json:"bridge_id,omitempty"`   // bridge id for external media snoop channel and external media channel.
	PlaybackID string `json:"playback_id,omitempty"` // playback id for reference channel's silence streaming out.

	ReferenceType ReferenceType `json:"reference_typee"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Status Status `json:"status,omitempty"` // status of the external media

	LocalIP   string `json:"local_ip"`
	LocalPort int    `json:"local_port"`

	ExternalHost    string        `json:"external_host"`
	Encapsulation   Encapsulation `json:"encapsulation"` // Payload encapsulation protocol
	Transport       Transport     `json:"transport"`
	ConnectionType  string        `json:"connection_type"`
	Format          string        `json:"format"`
	DirectionListen Direction     `json:"direction_listen,omitempty"` // direction of the external media channel, default is ""
	DirectionSpeak  Direction     `json:"direction_speak,omitempty"`  // direction of the external media channel, default is ""
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeCall       ReferenceType = "call"
	ReferenceTypeConfbridge ReferenceType = "confbridge"
)

// Encapsulation define
type Encapsulation string

// list of Encapsulation types
const (
	EncapsulationRTP         Encapsulation = "rtp"
	EncapsulationAudioSocket Encapsulation = "audiosocket"
)

// Transport define
type Transport string

// list of Transport types
const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
)

// Direction define
type Direction string

// list of direction types
const (
	DirectionNone Direction = ""     // no direction
	DirectionBoth Direction = "both" // both direction
	DirectionIn   Direction = "in"   // listen direction
	DirectionOut  Direction = "out"  // speak direction
)

type Status string

const (
	StatusRunning     Status = "running"     // running status
	StatusTerminating Status = "terminating" // terminating status
	StatusTerminated  Status = "terminated"  // terminated status
)

// Type define
type Type string

// list of Type types
const (
	TypeNormal    Type = "normal"
	TypeWebsocket Type = "websocket"
)
