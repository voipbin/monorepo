package bridge

import (
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
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

	// conference info
	ConferenceID   uuid.UUID
	ConferenceType conference.Type
	ConferenceJoin bool

	TMCreate string
	TMUpdate string
	TMDelete string
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

		ChannelIDs: e.Bridge.Channels,

		ConferenceID:   uuid.Nil,
		ConferenceType: conference.TypeNone,
		ConferenceJoin: false,

		TMCreate: string(e.Timestamp),
	}

	mapParse := parseBridgeName(b.Name)
	if mapParse["conference_id"] != "" {
		b.ConferenceID = uuid.FromStringOrNil(mapParse["conference_id"])
	}
	if mapParse["conference_type"] != "" {
		b.ConferenceType = conference.Type(mapParse["conference_type"])
	}
	if mapParse["join"] != "" {
		b.ConferenceJoin, _ = strconv.ParseBool(mapParse["join"])
	}

	return b
}

// parseBridgeName returns bridge name parse
func parseBridgeName(bridgeName string) map[string]string {
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

		ConferenceID:   uuid.Nil,
		ConferenceType: conference.TypeNone,
		ConferenceJoin: false,
	}

	mapParse := parseBridgeName(b.Name)
	if mapParse["conference_id"] != "" {
		b.ConferenceID = uuid.FromStringOrNil(mapParse["conference_id"])
	}
	if mapParse["conference_type"] != "" {
		b.ConferenceType = conference.Type(mapParse["conference_type"])
	}
	if mapParse["join"] != "" {
		b.ConferenceJoin, _ = strconv.ParseBool(mapParse["join"])
	}

	return b
}
