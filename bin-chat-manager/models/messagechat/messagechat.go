package messagechat

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
)

// Messagechat defines message for the chat
type Messagechat struct {
	commonidentity.Identity

	ChatID uuid.UUID `json:"chat_id" db:"chat_id,uuid"`

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
	TypeSystem Type = "system"
	TypeNormal Type = "normal"
)
