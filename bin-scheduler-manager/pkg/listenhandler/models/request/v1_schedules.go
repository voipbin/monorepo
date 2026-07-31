package request

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// V1DataSchedulesPost is
// v1 data type request struct for
// /v1/schedules POST
//
// The field set and json keys mirror the body marshaled by
// requesthandler.SchedulerV1ScheduleCreate exactly.
type V1DataSchedulesPost struct {
	CustomerID     uuid.UUID       `json:"customer_id"`
	Name           string          `json:"name"`
	Detail         string          `json:"detail"`
	Cron           string          `json:"cron"`
	TargetQueue    string          `json:"target_queue"`
	TargetURI      string          `json:"target_uri"`
	TargetMethod   string          `json:"target_method"`
	TargetDataType string          `json:"target_data_type"`
	TargetData     json.RawMessage `json:"target_data"`
	TimeoutMS      int             `json:"timeout_ms"`
	RetryMax       int             `json:"retry_max"`
	Enabled        bool            `json:"enabled"`
}

// V1DataSchedulesIDPut is
// v1 data type request struct for
// /v1/schedules/<schedule-id> PUT
//
// The client (requesthandler.SchedulerV1ScheduleUpdate) marshals a
// map[schedule.Field]any directly, e.g. {"cron":"0 4 * * *","enabled":true} —
// a flat object of only the fields being updated. Pointer fields distinguish
// "absent" from "zero value" so the listener can rebuild the partial field
// set (rpc.md §9.5).
type V1DataSchedulesIDPut struct {
	Name           *string          `json:"name,omitempty"`
	Detail         *string          `json:"detail,omitempty"`
	Cron           *string          `json:"cron,omitempty"`
	TargetQueue    *string          `json:"target_queue,omitempty"`
	TargetURI      *string          `json:"target_uri,omitempty"`
	TargetMethod   *string          `json:"target_method,omitempty"`
	TargetDataType *string          `json:"target_data_type,omitempty"`
	TargetData     *json.RawMessage `json:"target_data,omitempty"`
	TimeoutMS      *int             `json:"timeout_ms,omitempty"`
	RetryMax       *int             `json:"retry_max,omitempty"`
	Enabled        *bool            `json:"enabled,omitempty"`
}
