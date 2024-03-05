package messagechatroom

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
)

// Messagechatroom defines the message for the chatroom
type Messagechatroom struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	AgentID    uuid.UUID `json:"agent_id"`

	ChatroomID    uuid.UUID `json:"chatroom_id"`
	MessagechatID uuid.UUID `json:"messagechat_id"`

	// message defines
	Source *commonaddress.Address `json:"source"`
	Type   Type                   `json:"type"`
	Text   string                 `json:"text"`
	Medias []media.Media          `json:"medias"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeUnknown Type = ""
	TypeSystem  Type = "system"
	TypeNormal  Type = "normal"
)

// ConvertType converts the chat's type to the chatroom's type.
func ConvertType(chatType messagechat.Type) Type {
	switch chatType {
	case messagechat.TypeNormal:
		return TypeNormal

	case messagechat.TypeSystem:
		return TypeSystem

	default:
		return TypeUnknown
	}
}
