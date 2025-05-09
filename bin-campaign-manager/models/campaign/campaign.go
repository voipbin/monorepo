package campaign

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// Campaign defines
type Campaign struct {
	commonidentity.Identity

	Type Type `json:"type"`

	Execute Execute `json:"execute"` // if the execute is running, this sets to true

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status       Status    `json:"status"`
	ServiceLevel int       `json:"service_level"`
	EndHandle    EndHandle `json:"end_handle"`

	// action settings
	FlowID  uuid.UUID         `json:"flow_id"` // flow id for campaign execution
	Actions []fmaction.Action `json:"actions"` // this actions will be stored to the flow

	// resource info
	OutplanID      uuid.UUID `json:"outplan_id"`
	OutdialID      uuid.UUID `json:"outdial_id"`
	QueueID        uuid.UUID `json:"queue_id"`
	NextCampaignID uuid.UUID `json:"next_campaign_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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
