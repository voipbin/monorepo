package media

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Media defines
type Media struct {
	commonidentity.Identity

	Type     Type   `json:"type"`
	Filename string `json:"filename"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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
