package campaigncall

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Campaigncall defines
type Campaigncall struct {
	commonidentity.Identity

	CampaignID uuid.UUID `json:"campaign_id" db:"campaign_id,uuid"`

	OutplanID       uuid.UUID `json:"outplan_id" db:"outplan_id,uuid"`
	OutdialID       uuid.UUID `json:"outdial_id" db:"outdial_id,uuid"`
	OutdialTargetID uuid.UUID `json:"outdial_target_id" db:"outdial_target_id,uuid"`
	QueueID         uuid.UUID `json:"queue_id" db:"queue_id,uuid"`

	ActiveflowID uuid.UUID `json:"activeflow_id" db:"activeflow_id,uuid"` // activeflow id
	FlowID       uuid.UUID `json:"flow_id" db:"flow_id,uuid"`             // flow id

	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"` // none or call
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"` // reference id

	Status Status `json:"status" db:"status"`
	Result Result `json:"result" db:"result"`

	Source           *commonaddress.Address `json:"source" db:"source,json"`
	Destination      *commonaddress.Address `json:"destination" db:"destination,json"`
	DestinationIndex int                    `json:"destination_index" db:"destination_index"`
	TryCount         int                    `json:"try_count" db:"try_count"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
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
