package analysishistory

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/analysis"
)

// History is an append-only archived snapshot of a timeline analysis.
//
// A history row is written (in the same transaction as the live-row mutation)
// whenever a live analysis is superseded by a re-analyze or removed by a delete.
// It preserves the full lineage that the live table (one row per activeflow)
// cannot retain. History rows are never updated or deleted (append-only).
type History struct {
	commonidentity.Identity

	// AnalysisID is the live-table id this snapshot was archived from.
	AnalysisID   uuid.UUID `json:"analysis_id,omitempty" db:"analysis_id,uuid"`
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`

	Status analysis.Status `json:"status,omitempty" db:"status"`
	Result json.RawMessage `json:"result,omitempty" db:"result,json"`
	Model  string          `json:"model,omitempty" db:"model"`
	Error  string          `json:"error,omitempty" db:"error"`

	// Reason is why this snapshot was archived.
	Reason Reason `json:"reason,omitempty" db:"reason"`

	// TMCreate is when this history row was written (NOT the original analysis create time).
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// Reason is why an analysis snapshot was archived to history.
type Reason string

const (
	// ReasonReanalyze means the live row was reset to progressing for a re-analyze.
	ReasonReanalyze Reason = "reanalyze"
	// ReasonDelete means the live row was soft-deleted.
	ReasonDelete Reason = "delete"
)
