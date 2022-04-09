package campaign

import "github.com/gofrs/uuid"

type Campaign struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status       Status `json:"status"`
	ServiceLevel int    `json:"service_level"`

	// resource info
	OutplanID uuid.UUID `json:"outplan_id"`
	OutdialID uuid.UUID `json:"outdial_id"`
	QueueID   uuid.UUID `json:"queue_id"`

	NextCampaignID uuid.UUID `json:"next_campaign_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

type Status string

const (
	StatusStop     Status = "stop"
	StatusStopping Status = "stopping"
	StatusRun      Status = "run"
	StatusRunning  Status = "running"
	StatusPause    Status = "pause"
	StatusPausing  Status = "pausing"
)
