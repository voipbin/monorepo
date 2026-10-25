package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"
)

const timelineQueue = "bin-manager.timeline-manager.request"

func Test_TimelineV1AnalysisWrappers(t *testing.T) {
	analysisID := uuid.FromStringOrNil("f2a3b4c5-d6e7-4f80-9a1b-b3c4d5e6f708")
	customerID := uuid.FromStringOrNil("a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819")
	activeflowID := uuid.FromStringOrNil("b4c5d6e7-f809-4a1b-9c3d-d5e6f708192a")

	analysisWant := &tmanalysis.Analysis{
		Identity:     identity.Identity{ID: analysisID, CustomerID: customerID},
		ActiveflowID: activeflowID,
		Status:       tmanalysis.StatusCompleted,
	}
	const analysisResponse = `{"id":"f2a3b4c5-d6e7-4f80-9a1b-b3c4d5e6f708","customer_id":"a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819","activeflow_id":"b4c5d6e7-f809-4a1b-9c3d-d5e6f708192a","status":"completed"}`

	cases := []rpcCase{
		{
			name:           "create_reanalyze",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819","activeflow_id":"b4c5d6e7-f809-4a1b-9c3d-d5e6f708192a","reanalyze":true}`,
			responseData:   analysisResponse,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisCreate(context.Background(), customerID, activeflowID, true)
			},
			expectRes: analysisWant,
		},
		{
			name:           "create_first_time",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819","activeflow_id":"b4c5d6e7-f809-4a1b-9c3d-d5e6f708192a","reanalyze":false}`,
			responseData:   analysisResponse,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisCreate(context.Background(), customerID, activeflowID, false)
			},
			expectRes: analysisWant,
		},
		{
			name:           "get",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses/f2a3b4c5-d6e7-4f80-9a1b-b3c4d5e6f708?customer_id=a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   analysisResponse,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisGet(context.Background(), customerID, analysisID)
			},
			expectRes: analysisWant,
		},
		{
			// the page token is query escaped: space and plus must not survive as they are.
			name:           "list_with_filter_and_escaped_token",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses?customer_id=a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819&page_token=a+b%2Bc&page_size=20",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"status":"completed"}`,
			responseData:   `[` + analysisResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisList(context.Background(), customerID, "a b+c", 20, map[tmanalysis.Field]any{tmanalysis.FieldStatus: "completed"})
			},
			expectRes: []tmanalysis.Analysis{*analysisWant},
		},
		{
			name:           "list_without_filter",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses?customer_id=a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819&page_token=&page_size=0",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `null`,
			responseData:   `[]`,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisList(context.Background(), customerID, "", 0, nil)
			},
			expectRes: []tmanalysis.Analysis{},
		},
		{
			name:           "delete",
			queue:          timelineQueue,
			expectURI:      "/v1/analyses/f2a3b4c5-d6e7-4f80-9a1b-b3c4d5e6f708?customer_id=a3b4c5d6-e7f8-4091-8b2c-c4d5e6f70819",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   analysisResponse,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1AnalysisDelete(context.Background(), customerID, analysisID)
			},
			expectRes: analysisWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.TimelineV1AnalysisList(ctx, customerID, "", 10, map[tmanalysis.Field]any{tmanalysis.FieldStatus: make(chan int)})
	})
}
