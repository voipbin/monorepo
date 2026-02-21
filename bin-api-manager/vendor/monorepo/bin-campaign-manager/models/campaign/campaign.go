package campaign

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// Campaign defines
type Campaign struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	Execute Execute `json:"execute" db:"execute"` // if the execute is running, this sets to true

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	Status       Status    `json:"status" db:"status"`
	ServiceLevel int       `json:"service_level" db:"service_level"`
	EndHandle    EndHandle `json:"end_handle" db:"end_handle"`

	// action settings
	FlowID  uuid.UUID         `json:"flow_id" db:"flow_id,uuid"` // flow id for campaign execution
	Actions []fmaction.Action `json:"actions" db:"actions,json"` // this actions will be stored to the flow

	// resource info
	OutplanID      uuid.UUID `json:"outplan_id" db:"outplan_id,uuid"`
	OutdialID      uuid.UUID `json:"outdial_id" db:"outdial_id,uuid"`
	QueueID        uuid.UUID `json:"queue_id" db:"queue_id,uuid"`
	NextCampaignID uuid.UUID `json:"next_campaign_id" db:"next_campaign_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
	TypeCall Type = "call" // make a call to the destination
	TypeFlow Type = "flow" // execute a flow with the destination
)

// Execute defines
type Execute string

// list of executes
const (
	ExecuteRun  Execute = "run"
	ExecuteStop Execute = "stop"
)

// Status defines
type Status string

// list of Status
const (
	StatusStop     Status = "stop"
	StatusStopping Status = "stopping" // preparing stop
	StatusRun      Status = "run"
)

// EndHandle defines
type EndHandle string

// list of EndHandle types
const (
	EndHandleStop     EndHandle = "stop"     // the campaign will stop if the outdial has no more outdial target
	EndHandleContinue EndHandle = "continue" // the campaign will continue to run after outdial has no more outdial target.
)
