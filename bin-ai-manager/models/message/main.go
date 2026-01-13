package message

import (
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Message defines
type Message struct {
	identity.Identity

	AIcallID uuid.UUID `json:"aicall_id,omitempty" db:"aicall_id,uuid"`

	Direction Direction `json:"direction,omitempty" db:"direction"`
	Role      Role      `json:"role,omitempty" db:"role"`
	Content   string    `json:"content,omitempty" db:"content"`

	ToolCalls  []ToolCall `json:"tool_calls,omitempty" db:"tool_calls,json"`
	ToolCallID string     `json:"tool_call_id,omitempty" db:"tool_call_id"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

// Role defiens
type Role string

// list of roles
const (
	RoleNone      Role = ""
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
	DirectionNone     Direction = ""
)
