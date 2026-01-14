package outdial

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Outdial defines
type Outdial struct {
	commonidentity.Identity

	CampaignID uuid.UUID `json:"campaign_id" db:"campaign_id,uuid"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	Data string `json:"data" db:"data"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}
