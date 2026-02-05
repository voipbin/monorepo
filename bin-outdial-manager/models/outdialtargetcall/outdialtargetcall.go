package outdialtargetcall

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// OutdialTargetCall defines
type OutdialTargetCall struct {
	commonidentity.Identity

	CampaignID      uuid.UUID `json:"campaign_id" db:"campaign_id,uuid"`
	OutdialID       uuid.UUID `json:"outdial_id" db:"outdial_id,uuid"`
	OutdialTargetID uuid.UUID `json:"outdial_target_id" db:"outdial_target_id,uuid"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id" db:"activeflow_id,uuid"`   // this is required
	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`      // none or call
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`     // reference id

	Status Status `json:"status" db:"status"`

	Destination      *commonaddress.Address `json:"destination" db:"destination,json"`
	DestinationIndex int                    `json:"destination_index" db:"destination_index"`
	TryCount         int                    `json:"try_count" db:"try_count"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType defines
type ReferenceType string

// list of ReferenceTypes
const (
	ReferenceTypeNone = "none"
	ReferenceTypeCall = "call"
)

// Status defines
type Status string

// list of Status
const (
	StatusProgressing = "progressing"
	StatusDone        = "done"
)
