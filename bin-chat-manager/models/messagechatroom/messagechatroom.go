package messagechatroom

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
)

// Messagechatroom defines the message for the chatroom
type Messagechatroom struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"` //
	OwnerID    uuid.UUID `json:"owner_id"`

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

// OwnerType define
type OwnerType string

// list of owner types
const (
	OwnerTypeNonw  OwnerType = ""
	OwnerTypeAgent OwnerType = "agent"
)

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
