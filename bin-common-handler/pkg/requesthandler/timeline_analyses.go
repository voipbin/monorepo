package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TimelineV1AnalysisCreate sends a request to timeline-manager to create
// (or re-run) the analysis for an activeflow (POST /v1/analyses).
func (r *requestHandler) TimelineV1AnalysisCreate(ctx context.Context, customerID uuid.UUID, activeflowID uuid.UUID, reanalyze bool) (*tmanalysis.Analysis, error) {
	uri := "/v1/analyses"

	data := &struct {
		CustomerID   uuid.UUID `json:"customer_id"`
		ActiveflowID uuid.UUID `json:"activeflow_id"`
		Reanalyze    bool      `json:"reanalyze"`
	}{
		CustomerID:   customerID,
		ActiveflowID: activeflowID,
		Reanalyze:    reanalyze,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/analyses", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmanalysis.Analysis
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TimelineV1AnalysisGet sends a request to timeline-manager to get
// an analysis (GET /v1/analyses/<analysis-id>).
func (r *requestHandler) TimelineV1AnalysisGet(ctx context.Context, customerID uuid.UUID, analysisID uuid.UUID) (*tmanalysis.Analysis, error) {
	uri := fmt.Sprintf("/v1/analyses/%s?customer_id=%s", analysisID.String(), customerID.String())

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/analyses/<analysis-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmanalysis.Analysis
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TimelineV1AnalysisList sends a request to timeline-manager
// to getting a list of analyses info.
// it returns detail list of analyses info if it succeed.
func (r *requestHandler) TimelineV1AnalysisList(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[tmanalysis.Field]any) ([]tmanalysis.Analysis, error) {
	uri := fmt.Sprintf("/v1/analyses?customer_id=%s&page_token=%s&page_size=%d", customerID.String(), url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/analyses", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []tmanalysis.Analysis
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// TimelineV1AnalysisDelete sends a request to timeline-manager
// to deleting an analysis.
// it returns deleted analysis if it succeed.
func (r *requestHandler) TimelineV1AnalysisDelete(ctx context.Context, customerID uuid.UUID, analysisID uuid.UUID) (*tmanalysis.Analysis, error) {
	uri := fmt.Sprintf("/v1/analyses/%s?customer_id=%s", analysisID.String(), customerID.String())

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodDelete, "timeline/analyses/<analysis-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmanalysis.Analysis
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
