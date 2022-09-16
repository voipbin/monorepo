package messagechat

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
)

// Messagechat defines message for the chat
type Messagechat struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ChatID uuid.UUID `json:"chat_id"`

	// message defines
	Source *commonaddress.Address
	Type   Type
	Text   string
	Medias []media.Media

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeSystem Type = "system"
	TypeNormal Type = "normal"
)
