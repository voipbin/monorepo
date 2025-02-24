package message

import (
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Message defines
type Message struct {
	identity.Identity
	ChatbotcallID uuid.UUID `json:"chatbotcall_id,omitempty"`

	Direction Direction `json:"direction,omitempty"`
	Role      Role      `json:"role,omitempty"`
	Content   string    `json:"content,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// Role defiens
type Role string

// list of roles
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
	RoleTool      Role = "tool"
)

type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)
