package resource

import "github.com/gofrs/uuid"

// Resource defines
type Resource struct {
	ID            uuid.UUID     `json:"id"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	OwnerID       uuid.UUID     `json:"owner_id"`
	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Data interface{} `json:"data"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ReferenceType defines
type ReferenceType string

// list of Reference types
const (
	ReferenceTypeCall            ReferenceType = "call"            // call-manager's call
	ReferenceTypeGroupcall       ReferenceType = "groupcall"       // call-manager's groupcall
	ReferenceTypeConversation    ReferenceType = "conversation"    // conversation-manager's conversation
	ReferenceTypeChatroom        ReferenceType = "chatroom"        // chat-manager's chatroom
	ReferenceTypeMessagechatroom ReferenceType = "messagechatroom" // chat-manager's messagechatroom
)
