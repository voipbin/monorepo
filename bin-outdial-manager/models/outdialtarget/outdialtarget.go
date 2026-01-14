package outdialtarget

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// OutdialTarget defines
type OutdialTarget struct {
	ID        uuid.UUID `json:"id" db:"id,uuid"`
	OutdialID uuid.UUID `json:"outdial_id" db:"outdial_id,uuid"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	Data   string `json:"data" db:"data"`
	Status Status `json:"status" db:"status"`

	// destinations
	Destination0 *commonaddress.Address `json:"destination_0" db:"destination_0,json"` // destination address 0
	Destination1 *commonaddress.Address `json:"destination_1" db:"destination_1,json"` // destination address 1
	Destination2 *commonaddress.Address `json:"destination_2" db:"destination_2,json"` // destination address 2
	Destination3 *commonaddress.Address `json:"destination_3" db:"destination_3,json"` // destination address 3
	Destination4 *commonaddress.Address `json:"destination_4" db:"destination_4,json"` // destination address 4

	// try counts
	TryCount0 int `json:"try_count_0" db:"try_count_0"` // try count for destination 0
	TryCount1 int `json:"try_count_1" db:"try_count_1"` // try count for destination 1
	TryCount2 int `json:"try_count_2" db:"try_count_2"` // try count for destination 2
	TryCount3 int `json:"try_count_3" db:"try_count_3"` // try count for destination 3
	TryCount4 int `json:"try_count_4" db:"try_count_4"` // try count for destination 4

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// Status defines
type Status string

// list of status
const (
	StatusProgressing Status = "progressing"
	StatusDone        Status = "done"
	StatusIdle        Status = "idle"
)
