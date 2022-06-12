package message

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// Message defines
type Message struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ConversationID uuid.UUID `json:"conversation_id"`
	Status         Status    `json:"status"`

	ReferenceType conversation.ReferenceType `json:"reference_type"` // used for find a conversation info(source info: group/room/user)
	ReferenceID   string                     `json:"reference_id"`   // used for find a conversation info(source info: group_id, room_id, user_id)

	SourceTarget string `json:"source_target"` // message source target.

	Type Type   `json:"type"`
	Data []byte `json:"message"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status defines
type Status string

// list of Status
const (
	StatusSending  Status = "sending"
	StatusSent     Status = "sent"
	StatusFailed   Status = "failed"
	StatusReceived Status = "received"
)

// Type defines
type Type string

// list of types
const (
	TypeText     Type = "text"
	TypeImage    Type = "image"
	TypeVideo    Type = "video"
	TypeAudio    Type = "audio"
	TypeFile     Type = "file"
	TypeLocation Type = "location"
	TypeSticker  Type = "sticker"
	TypeTemplate Type = "template"
	TypeImagemap Type = "imagemap"
	TypeFlex     Type = "flex"
)

// Matches return true if the given items are the same
func (h *Message) Matches(x interface{}) bool {
	comp := x.(*Message)
	c := *h

	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate
	c.TMDelete = comp.TMDelete

	return reflect.DeepEqual(c, *comp)
}

func (h *Message) String() string {
	return fmt.Sprintf("%v", *h)
}
