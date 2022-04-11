package outplan

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Outplan defines
type Outplan struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	// basic info
	Name   string `json:"name"`
	Detail string `json:"detail"`

	// action settings
	FlowID    uuid.UUID          `json:"flow_id"`
	Actions   []action.Action    `json:"actions"`
	Source    *cmaddress.Address `json:"source"` // caller id
	EndHandle EndHandle          `json:"end_handle"`

	// plan dial settings
	DialTimeout  int `json:"dial_timeout"` // milliseconds
	TryInterval  int `json:"try_interval"` // milliseconds
	MaxTryCount0 int `json:"max_try_count_0"`
	MaxTryCount1 int `json:"max_try_count_1"`
	MaxTryCount2 int `json:"max_try_count_2"`
	MaxTryCount3 int `json:"max_try_count_3"`
	MaxTryCount4 int `json:"max_try_count_4"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// const defines
const (
	MaxTryCountLen = 5 // length of max try count
)

// EndHandle defines
type EndHandle string

// list of EndHandle types
const (
	EndHandleStop     EndHandle = "stop"
	EndHandleContinue EndHandle = "continue"
)
