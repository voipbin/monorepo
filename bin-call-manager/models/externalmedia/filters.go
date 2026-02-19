package externalmedia

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for ExternalMedia queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	AsteriskID    string        `filter:"asterisk_id"`
	ChannelID     string        `filter:"channel_id"`
	BridgeID      string        `filter:"bridge_id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	ReferenceID   uuid.UUID     `filter:"reference_id"`
	Status        Status        `filter:"status"`
	Type          Type          `filter:"type"`
}
