package outdialtarget

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// OutdialTarget defines
type OutdialTarget struct {
	ID        uuid.UUID `json:"id"`
	OutdialID uuid.UUID `json:"outdial_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Data   string `json:"data"`
	Status Status `json:"status"`

	// destinations
	Destination0 *commonaddress.Address `json:"destination_0"` // destination address 0
	Destination1 *commonaddress.Address `json:"destination_1"` // destination address 1
	Destination2 *commonaddress.Address `json:"destination_2"` // destination address 2
	Destination3 *commonaddress.Address `json:"destination_3"` // destination address 3
	Destination4 *commonaddress.Address `json:"destination_4"` // destination address 4

	// try counts
	TryCount0 int `json:"try_count_0"` // try count for destination 0
	TryCount1 int `json:"try_count_1"` // try count for destination 1
	TryCount2 int `json:"try_count_2"` // try count for destination 2
	TryCount3 int `json:"try_count_3"` // try count for destination 3
	TryCount4 int `json:"try_count_4"` // try count for destination 4

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status defines
type Status string

// list of status
const (
	StatusProgressing Status = "progressing"
	StatusDone        Status = "done"
	StatusIdle        Status = "idle"
)
