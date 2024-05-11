package media

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// Media define
type Media struct {
	Type    Type                  `json:"type,omitempty"`
	Address commonaddress.Address `json:"address,omitempty"`  // valid only if the Type is address type
	FileID  uuid.UUID             `json:"file_id,omitempty"`  // valid only if the Type is file
	LinkURL string                `json:"link_url,omitempty"` // valid only if the Type is link type
}

// Type define
type Type string

// list of types
const (
	TypeAddress Type = "address" // the media contains address info
	TypeFile    Type = "file"    // the media contains file info
	TypeLink    Type = "link"    // the media contains link info
)
