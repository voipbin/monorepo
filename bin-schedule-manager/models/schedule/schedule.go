package schedule

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type defines the schedule's execution type.
type Type string

// list of schedule types
const (
	TypeRPC Type = "rpc"
)

// Schedule data model
type Schedule struct {
	commonidentity.Identity

	Name   string `json:"name" db:"name"`     // schedule's name
	Detail string `json:"detail" db:"detail"` // schedule's detail

	Type Type   `json:"type" db:"type"` // schedule's execution type
	Cron string `json:"cron" db:"cron"` // 5-field cron expression, evaluated in UTC

	TargetQueue    string          `json:"target_queue" db:"target_queue"`         // target request queue name
	TargetURI      string          `json:"target_uri" db:"target_uri"`             // target request uri
	TargetMethod   string          `json:"target_method" db:"target_method"`       // target request method
	TargetDataType string          `json:"target_data_type" db:"target_data_type"` // target request data type
	TargetData     json.RawMessage `json:"target_data" db:"target_data,json"`      // target request payload

	TimeoutMS int  `json:"timeout_ms" db:"timeout_ms"` // RPC timeout(ms) for the dispatch
	RetryMax  int  `json:"retry_max" db:"retry_max"`   // in-run immediate retries on failure
	Enabled   bool `json:"enabled" db:"enabled"`

	TMNextRun *time.Time `json:"tm_next_run" db:"tm_next_run"` // Next run timestamp. NULL = compute on next scan.
	TMLastRun *time.Time `json:"tm_last_run" db:"tm_last_run"` // Last successful claim timestamp.

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // Deleted timestamp.
}
