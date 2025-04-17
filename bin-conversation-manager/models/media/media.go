package media

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Media defines
type Media struct {
	commonidentity.Identity

	Type     Type   `json:"type,omitempty"`
	Filename string `json:"filename,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
