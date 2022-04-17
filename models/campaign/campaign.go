package campaign

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

type Campaign struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status       Status    `json:"status"`
	ServiceLevel int       `json:"service_level"`
	EndHandle    EndHandle `json:"end_handle"`

	// action settings
	FlowID  uuid.UUID         `json:"flow_id"`
	Actions []fmaction.Action `json:"actions"`

	// resource info
	OutplanID uuid.UUID `json:"outplan_id"`
	OutdialID uuid.UUID `json:"outdial_id"`
	QueueID   uuid.UUID `json:"queue_id"`

	NextCampaignID uuid.UUID `json:"next_campaign_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status defines
type Status string

// list of Status
const (
	StatusStop     Status = "stop"
	StatusStopping Status = "stopping" // preparing stop
	StatusRun      Status = "run"
	StatusRunning  Status = "running" // preparing run
	// StatusPause    Status = "pause"
	// StatusPausing  Status = "pausing" // preparing pause
)

// EndHandle defines
type EndHandle string

// list of EndHandle types
const (
	EndHandleStop     EndHandle = "stop"     // the campaign will stop if the outdial has no more outdial target
	EndHandleContinue EndHandle = "continue" // the campaign will continue to run after outdial has no more outdial target.
)
