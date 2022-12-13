package bridge

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
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

	ChannelIDs []string

	// reference
	ReferenceType ReferenceType
	ReferenceID   uuid.UUID

	TMCreate string
	TMUpdate string
	TMDelete string
}

// Matches return true if the given items are the same
func (a *Bridge) Matches(x interface{}) bool {
	comp := x.(*Bridge)
	c := *a

	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate
	c.TMDelete = comp.TMDelete

	return reflect.DeepEqual(c, *comp)
}

func (a *Bridge) String() string {
	return fmt.Sprintf("%v", *a)
}

// Tech type
type Tech string

// List of Tech types
const (
	TechSimple  Tech = "simple_bridge"
	TechSoftmix Tech = "softmix"
)

// Type shows bridge's type
type Type string

// List of types
const (
	TypeMixing     Type = "mixing"
	TypeDTMFEvents Type = "dtmf_events"
	TypeProxyMedia Type = "proxy_media"
	TypeHolding    Type = "holding"
	TypeVideoSFU   Type = "video_sfu"
)

// ReferenceType defines
type ReferenceType string

// List of Reference types
const (
	ReferenceTypeUnknown         ReferenceType = "unknown"
	ReferenceTypeCall            ReferenceType = "call"       // call bridge
	ReferenceTypeCallSnoop       ReferenceType = "call-snoop" // snoop bridge for the call. usually used by the spy channel to the call
	ReferenceTypeConfbridge      ReferenceType = "confbridge"
	ReferenceTypeConfbridgeSnoop ReferenceType = "confbridge-snoop" // snoop bridge for the confbridge. usually used by the spy channel to the confbridge bridge
)

// NewBridgeByBridgeCreated creates Bridge based on ARI BridgeCreated event
func NewBridgeByBridgeCreated(e *ari.BridgeCreated) *Bridge {

	b := NewBridgeByARIBridge(&e.Bridge)
	b.AsteriskID = e.AsteriskID
	b.TMCreate = string(e.Timestamp)

	return b
}

// ParseBridgeName returns bridge name parse
func ParseBridgeName(bridgeName string) map[string]string {
	res := map[string]string{}

	if bridgeName == "" {
		return res
	}

	pairs := strings.Split(bridgeName, ",")
	for _, pair := range pairs {
		tmp := strings.Split(pair, "=")
		if len(tmp) != 2 {
			continue
		}
		res[tmp[0]] = tmp[1]
	}

	return res
}

// NewBridgeByARIBridge returns partial of bridge struct
func NewBridgeByARIBridge(e *ari.Bridge) *Bridge {

	b := &Bridge{
		ID:   e.ID,
		Name: e.Name,

		// info
		Type: Type(e.BridgeType),
		Tech: Tech(e.Technology),

		Class:   e.BridgeClass,
		Creator: e.Creator,

		VideoMode:     e.VideoMode,
		VideoSourceID: e.VideoSourceID,

		ChannelIDs: e.Channels,

		ReferenceType: ReferenceTypeUnknown,
		ReferenceID:   uuid.Nil,
	}

	mapParse := ParseBridgeName(b.Name)

	// reference info
	if mapParse["reference_type"] != "" {
		b.ReferenceType = ReferenceType(mapParse["reference_type"])
	}
	b.ReferenceID = uuid.FromStringOrNil(mapParse["reference_id"])

	return b
}
