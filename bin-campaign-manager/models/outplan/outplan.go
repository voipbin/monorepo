package outplan

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// Outplan defines
type Outplan struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	// basic info
	Name   string `json:"name"`
	Detail string `json:"detail"`

	// source settings
	Source *commonaddress.Address `json:"source"` // caller id

	// plan dial settings
	DialTimeout int `json:"dial_timeout"` // milliseconds
	TryInterval int `json:"try_interval"` // milliseconds

	// max try count
	MaxTryCount0 int `json:"max_try_count_0"` // max try count for destination_0
	MaxTryCount1 int `json:"max_try_count_1"` // max try count for destination_1
	MaxTryCount2 int `json:"max_try_count_2"` // max try count for destination_2
	MaxTryCount3 int `json:"max_try_count_3"` // max try count for destination_3
	MaxTryCount4 int `json:"max_try_count_4"` // max try count for destination_4

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// const defines
const (
	MaxTryCountLen = 5 // length of max try count
)
