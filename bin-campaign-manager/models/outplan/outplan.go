package outplan

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Outplan defines
type Outplan struct {
	commonidentity.Identity

	// basic info
	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	// source settings
	Source *commonaddress.Address `json:"source" db:"source,json"` // caller id

	// plan dial settings
	DialTimeout int `json:"dial_timeout" db:"dial_timeout"` // milliseconds
	TryInterval int `json:"try_interval" db:"try_interval"` // milliseconds

	// max try count
	MaxTryCount0 int `json:"max_try_count_0" db:"max_try_count_0"` // max try count for destination_0
	MaxTryCount1 int `json:"max_try_count_1" db:"max_try_count_1"` // max try count for destination_1
	MaxTryCount2 int `json:"max_try_count_2" db:"max_try_count_2"` // max try count for destination_2
	MaxTryCount3 int `json:"max_try_count_3" db:"max_try_count_3"` // max try count for destination_3
	MaxTryCount4 int `json:"max_try_count_4" db:"max_try_count_4"` // max try count for destination_4

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// const defines
const (
	MaxTryCountLen = 5 // length of max try count
)
