package campaigncall

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// Campaigncall defines
type Campaigncall struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	CampaignID uuid.UUID `json:"campaign_id"`

	OutplanID       uuid.UUID `json:"outplan_id"`
	OutdialID       uuid.UUID `json:"outdial_id"`
	OutdialTargetID uuid.UUID `json:"outdial_target_id"`
	QueueID         uuid.UUID `json:"queue_id"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id"`  // this is required
	ReferenceType ReferenceType `json:"reference_type"` // none or call
	ReferenceID   uuid.UUID     `json:"reference_id"`   // reference id

	Status Status `json:"status"`

	Source           *cmaddress.Address `json:"source"`
	Destination      *cmaddress.Address `json:"destination"`
	DestinationIndex int                `json:"destination_index"`
	TryCount         int                `json:"try_count"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
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
	StatusDialing     = "dialing"     // the campaigncall is dialing(not answered yet)
	StatusProgressing = "progressing" // the campaigncall is progressing(the call answered)
	StatusDone        = "done"        // the campaigncall is hungup
)
