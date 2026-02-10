package extensiondirect

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// ExtensionDirect struct
type ExtensionDirect struct {
	commonidentity.Identity

	ExtensionID uuid.UUID `json:"extension_id" db:"extension_id,uuid"`
	Hash        string    `json:"hash" db:"hash"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
