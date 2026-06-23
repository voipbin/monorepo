package analysishandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
)

// Get returns an analysis by id, ownership-checked. A cross-customer or absent
// record returns the SAME masked ErrNotFound (no existence oracle — review C1).
func (h *analysisHandler) Get(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error) {
	res, err := h.dbHandler.AnalysisGet(ctx, customerID, id)
	if err != nil {
		if errors.Is(err, analysisdbhandler.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("Get: could not get analysis. err: %v", err)
	}
	return res, nil
}

// GetByActiveflowID returns the live analysis for an activeflow, ownership-checked.
func (h *analysisHandler) GetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error) {
	res, err := h.dbHandler.AnalysisGetByActiveflowID(ctx, customerID, activeflowID)
	if err != nil {
		if errors.Is(err, analysisdbhandler.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetByActiveflowID: could not get analysis. err: %v", err)
	}
	return res, nil
}

// List returns a paginated list, always filtered by customer_id (the authority).
func (h *analysisHandler) List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[analysis.Field]any) ([]*analysis.Analysis, error) {
	res, err := h.dbHandler.AnalysisList(ctx, customerID, pageSize, pageToken, filters)
	if err != nil {
		return nil, fmt.Errorf("List: could not list analyses. err: %v", err)
	}
	return res, nil
}

// Delete soft-deletes (archive-then-delete), ownership-checked.
func (h *analysisHandler) Delete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error) {
	res, err := h.dbHandler.AnalysisArchiveAndDelete(ctx, customerID, id)
	if err != nil {
		if errors.Is(err, analysisdbhandler.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("Delete: could not delete analysis. err: %v", err)
	}
	return res, nil
}
