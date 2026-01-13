package messagechatroom

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
)

// Messagechatroom defines the message for the chatroom
type Messagechatroom struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatroomID    uuid.UUID `json:"chatroom_id" db:"chatroom_id,uuid"`
	MessagechatID uuid.UUID `json:"messagechat_id" db:"messagechat_id,uuid"`

	// message defines
	Source *commonaddress.Address `json:"source" db:"source,json"`
	Type   Type                   `json:"type" db:"type"`
	Text   string                 `json:"text" db:"text"`
	Medias []media.Media          `json:"medias" db:"medias,json"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
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
