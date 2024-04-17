package campaigncall

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
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

	ActiveflowID uuid.UUID `json:"activeflow_id"` // activeflow id
	FlowID       uuid.UUID `json:"flow_id"`       // flow id

	ReferenceType ReferenceType `json:"reference_type"` // none or call
	ReferenceID   uuid.UUID     `json:"reference_id"`   // reference id

	Status Status `json:"status"`
	Result Result `json:"result"`

	Source           *commonaddress.Address `json:"source"`
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
	ReferenceTypeNone ReferenceType = "none"
	ReferenceTypeCall ReferenceType = "call"
	ReferenceTypeFlow ReferenceType = "flow"
)

// Status defines
type Status string

// list of Status
const (
	StatusDialing     Status = "dialing"     // the campaigncall is dialing(not answered yet)
	StatusProgressing Status = "progressing" // the campaigncall is progressing(the call answered)
	StatusDone        Status = "done"        // the campaigncall is hungup
)

// Result defines
type Result string

// list of results
const (
	ResultNone    Result = ""        // no result.
	ResultSuccess Result = "success" // success
	ResultFail    Result = "fail"    // fail
)
