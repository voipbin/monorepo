package resource

import "github.com/gofrs/uuid"

// Resource defines
type Resource struct {
	ID            uuid.UUID `json:"id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	AgentID       uuid.UUID `json:"agent_id"`
	ReferenceType Type      `json:"reference_type"`
	ReferenceID   uuid.UUID `json:"reference_id"`

	Data interface{} `json:"data"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

type Type string

const (
	TypeCall             Type = "call"         // call-manager's call
	TypeGroupcall        Type = "groupcall"    // call-manager's groupcall
	TypeConversationType Type = "conversation" // conversation-manager's conversation
)
