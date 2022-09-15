package message

import (
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
)

// Message define
type Message struct {
	Source *commonaddress.Address `json:"source"`

	Type Type `json:"type"`

	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`
}

// Type define
type Type string

// list of types
const (
	TypeSystem Type = "system"
	TypeNormal Type = "normal"
)
