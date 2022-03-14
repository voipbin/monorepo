package message

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
)

// Message defines
// used only for the swag.
type Message struct {
	ID   uuid.UUID      `json:"id"`
	Type mmmessage.Type `json:"type"`

	// from/to info
	Source  *cmaddress.Address `json:"source"`
	Targets []target.Target    `json:"targets"`

	// message info
	Text string `json:"text"` // Text delivered in the body of the message.

	Direction mmmessage.Direction `json:"direction"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
