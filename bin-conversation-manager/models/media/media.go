package media

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Media defines
type Media struct {
	commonidentity.Identity

	Type     Type   `json:"type,omitempty" db:"type"`
	Filename string `json:"filename,omitempty" db:"filename"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
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
