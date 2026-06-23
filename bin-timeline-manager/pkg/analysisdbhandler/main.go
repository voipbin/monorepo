package analysisdbhandler

//go:generate mockgen -package analysisdbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/models/analysishistory"
)

// ErrNotFound is returned when a requested analysis record does not exist
// (or is not owned by the requesting customer — same masked error, no oracle).
var ErrNotFound = errors.New("record not found")

// ErrDuplicate is returned by AnalysisCreate on a UNIQUE(activeflow_id) conflict.
var ErrDuplicate = errors.New("duplicate live analysis for activeflow")

// AnalysisDBHandler is the MySQL persistence for timeline analyses.
//
// timeline-manager's primary store is ClickHouse (events); this is a SECOND,
// distinct persistence engine for the single mutable per-activeflow analysis
// record plus its append-only history. It is intentionally a separate handler
// package so the ClickHouse dbhandler stays untouched.
//
// Two tables:
//   - timeline_analyses          : one live row per activeflow, UNIQUE(activeflow_id).
//   - timeline_analysis_histories: append-only archive written (in the same txn)
//     before a re-analyze reset or a delete, preserving the superseded snapshot.
type AnalysisDBHandler interface {
	// AnalysisCreate inserts a fresh progressing row. On a UNIQUE(activeflow_id)
	// violation it returns ErrDuplicate so the caller can return the in-flight row.
	AnalysisCreate(ctx context.Context, a *analysis.Analysis) error
	// AnalysisGet returns an analysis by id, ownership-checked (masked not-found).
	AnalysisGet(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)
	// AnalysisGetByActiveflowID returns the live analysis for an activeflow, ownership-checked.
	AnalysisGetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error)
	// AnalysisList returns a paginated list (always filtered by customer_id).
	AnalysisList(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[analysis.Field]any) ([]*analysis.Analysis, error)
	// AnalysisArchiveAndReset archives the current non-progressing live row to
	// history (reason='reanalyze') then flips it back to progressing, atomically
	// in ONE transaction (CAS on status!='progressing'). Returns rowsAffected of
	// the reset UPDATE: 0 means another reanalyze already won (txn rolled back).
	AnalysisArchiveAndReset(ctx context.Context, id uuid.UUID) (int64, error)
	// AnalysisUpdateResult writes the final result (completed/failed). Guards on
	// status='progressing' AND tm_delete IS NULL so a deleted/swept row is not
	// resurrected. Returns rowsAffected.
	AnalysisUpdateResult(ctx context.Context, id uuid.UUID, status analysis.Status, result []byte, model, errStr string) (int64, error)
	// AnalysisArchiveAndDelete archives the current live row to history
	// (reason='delete') then soft-deletes it, ownership-checked, atomically in
	// ONE transaction. Returns the deleted record.
	AnalysisArchiveAndDelete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)

	// AnalysisHistoryList returns the append-only history for an activeflow,
	// newest first, always filtered by customer_id.
	AnalysisHistoryList(ctx context.Context, customerID, activeflowID uuid.UUID, size uint64, token string) ([]*analysishistory.History, error)

	// AnalysisCountProgressing returns the number of in-flight (progressing,
	// non-deleted) analyses for a customer. Used by the per-customer concurrency
	// cap (design F1).
	AnalysisCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error)
}

type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
}

// NewAnalysisDBHandler creates a new MySQL-backed analysis db handler.
func NewAnalysisDBHandler(db *sql.DB) AnalysisDBHandler {
	return &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
	}
}
