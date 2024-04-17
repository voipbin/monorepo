package outdialtargetcall

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// OutdialTargetCall defines
type OutdialTargetCall struct {
	ID              uuid.UUID `json:"id"`
	CustomerID      uuid.UUID `json:"customer_id"`
	CampaignID      uuid.UUID `json:"campaign_id"`
	OutdialID       uuid.UUID `json:"outdial_id"`
	OutdialTargetID uuid.UUID `json:"outdial_target_id"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id"`  // this is required
	ReferenceType ReferenceType `json:"reference_type"` // none or call
	ReferenceID   uuid.UUID     `json:"reference_id"`   // reference id

	Status Status `json:"status"`

	Destination      *commonaddress.Address `json:"destination"`
	DestinationIndex int                    `json:"destination_index"`
	TryCount         int                    `json:"try_count"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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
