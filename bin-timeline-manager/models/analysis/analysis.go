package analysis

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Analysis is the persisted AI analysis of an ended activeflow.
//
// One live row per activeflow (UNIQUE(activeflow_id, tm_delete)). The structured
// verdict lives in Result (a versioned JSON document, see models/verdict). The
// analysis is produced on demand, stored once, and may be manually re-analyzed
// (overwrite in place).
type Analysis struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`

	Status Status          `json:"status,omitempty" db:"status"`
	Result json.RawMessage `json:"result,omitempty" db:"result,json"`
	Model  string          `json:"model,omitempty" db:"model"`
	Error  string          `json:"error,omitempty" db:"error"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Status is the analysis lifecycle state. No StatusNone="" (zero-value hazard).
type Status string

const (
	// StatusProgressing means the analysis chain is running.
	StatusProgressing Status = "progressing"
	// StatusCompleted means the structured verdict has been persisted.
	StatusCompleted Status = "completed"
	// StatusFailed means the chain failed; Error carries a sanitized reason.
	StatusFailed Status = "failed"
)
