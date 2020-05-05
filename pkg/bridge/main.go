package bridge

import (
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

// Bridge struct represent asterisk's bridge information
type Bridge struct {
	// identity
	AsteriskID string
	ID         string
	Name       string

	// info
	Type    Type
	Tech    Tech
	Class   string
	Creator string

	VideoMode     string
	VideoSourceID string

	Channels []string

	TMCreate string
	TMUpdate string
	TMDelete string
}

// Tech type
type Tech string

// List of Tech types
const (
	TechSimple Tech = "simple_bridge"
)

// Type shows bridge's type
type Type string

// List of types
const (
	TypeMixing     Type = "mixing"
	TypeDTMFEvents Type = "demf_events"
	TypeProxyMedia Type = "proxy_media"
	TypeHolding    Type = "holding"
	TypeVideoSFU   Type = "video_sfu"
)

// NewBridgeByBridgeCreated creates Bridge based on ARI BridgeCreated event
func NewBridgeByBridgeCreated(e *ari.BridgeCreated) *Bridge {
	b := &Bridge{
		AsteriskID: e.AsteriskID,
		ID:         e.Bridge.ID,
		Name:       e.Bridge.Name,

		// info
		Type: Type(e.Bridge.BridgeType),
		Tech: Tech(e.Bridge.Technology),

		Class:   e.Bridge.BridgeClass,
		Creator: e.Bridge.Creator,

		VideoMode:     e.Bridge.VideoMode,
		VideoSourceID: e.Bridge.VideoSourceID,

		Channels: e.Bridge.Channels,

		TMCreate: string(e.Timestamp),
	}

	return b
}
