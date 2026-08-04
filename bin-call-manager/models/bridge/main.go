package bridge

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
)

// Bridge struct represent asterisk's bridge information
//
// db tag table (VOIP-1078 squirrel migration, plan §5 B-3 / R-D):
//
//	Go field       | column          | conversion
//	---------------+-----------------+-----------
//	AsteriskID     | asterisk_id     | -
//	ID             | id              | -          (varchar(255), NOT a uuid column)
//	Name           | name            | -
//	Type           | type            | -
//	Tech           | tech            | -
//	Class          | class           | -
//	Creator        | creator         | -
//	VideoMode      | video_mode      | -
//	VideoSourceID  | video_source_id | -
//	ChannelIDs     | channel_ids     | json       (mandatory: createScanPtr needs it for slices)
//	ReferenceType  | reference_type  | -
//	ReferenceID    | reference_id    | uuid       (BINARY(16); without ,uuid the write path
//	               |                 |             sends a 36-char string, see plan R-C/R-D)
//	TMCreate       | tm_create       | -          (*time.Time handled by ScanRow)
//	TMUpdate       | tm_update       | -
//	TMDelete       | tm_delete       | -
type Bridge struct {
	// identity
	AsteriskID string `db:"asterisk_id"`
	ID         string `db:"id"`
	Name       string `db:"name"`

	// info
	Type    Type   `db:"type"`
	Tech    Tech   `db:"tech"`
	Class   string `db:"class"`
	Creator string `db:"creator"`

	VideoMode     string `db:"video_mode"`
	VideoSourceID string `db:"video_source_id"`

	ChannelIDs []string `db:"channel_ids,json"`

	// reference
	ReferenceType ReferenceType `db:"reference_type"`
	ReferenceID   uuid.UUID     `db:"reference_id,uuid"`

	TMCreate *time.Time `db:"tm_create"`
	TMUpdate *time.Time `db:"tm_update"`
	TMDelete *time.Time `db:"tm_delete"`
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
	b.TMCreate = e.Timestamp.ToTime()

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
