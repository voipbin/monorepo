package listenhandler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysishandler"
)

func newAnalysisListenHandler(t *testing.T) (*listenHandler, *analysishandler.MockAnalysisHandler, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockAnalysis := analysishandler.NewMockAnalysisHandler(ctrl)
	h := &listenHandler{analysisHandler: mockAnalysis}
	return h, mockAnalysis, ctrl
}

func Test_v1AnalysesPost_ok(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	body, _ := json.Marshal(map[string]any{
		"customer_id":   cust.String(),
		"activeflow_id": af.String(),
		"reanalyze":     false,
	})
	m := &sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: body}

	row := &analysis.Analysis{Status: analysis.StatusProgressing}
	mockAnalysis.EXPECT().Start(gomock.Any(), cust, af, false).Return(row, nil)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func Test_v1AnalysesPost_cooldown_429(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	body, _ := json.Marshal(map[string]any{
		"customer_id":   cust.String(),
		"activeflow_id": af.String(),
		"reanalyze":     true,
	})
	m := &sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: body}

	mockAnalysis.EXPECT().Start(gomock.Any(), cust, af, true).Return(nil, analysishandler.ErrReanalyzeCooldown)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", res.StatusCode)
	}
}

func Test_v1AnalysesPost_not_ended_409(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	body, _ := json.Marshal(map[string]any{
		"customer_id":   cust.String(),
		"activeflow_id": af.String(),
	})
	m := &sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: body}

	mockAnalysis.EXPECT().Start(gomock.Any(), cust, af, false).Return(nil, analysishandler.ErrActiveflowNotEnded)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

func Test_v1AnalysesIDGet_masked_404(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	m := &sock.Request{
		URI:    "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(),
		Method: sock.RequestMethodGet,
	}

	mockAnalysis.EXPECT().Get(gomock.Any(), cust, id).Return(nil, analysishandler.ErrNotFound)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

func Test_v1AnalysesIDDelete_ok(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	m := &sock.Request{
		URI:    "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(),
		Method: sock.RequestMethodDelete,
	}

	row := &analysis.Analysis{Status: analysis.StatusCompleted}
	mockAnalysis.EXPECT().Delete(gomock.Any(), cust, id).Return(row, nil)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func Test_v1AnalysesGet_list(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	m := &sock.Request{
		URI:    "/v1/analyses?customer_id=" + cust.String() + "&page_size=10",
		Method: sock.RequestMethodGet,
	}

	mockAnalysis.EXPECT().
		List(gomock.Any(), cust, "", uint64(10), gomock.Any()).
		Return([]*analysis.Analysis{{ActiveflowID: af}}, nil)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	// The list response MUST be a bare JSON array, matching the platform-majority
	// list contract that the bin-common-handler requesthandler unmarshals into
	// []analysis.Analysis. Emitting an object wrapper ({result,next_page_token})
	// here is what caused the VOIP-1199 "cannot unmarshal object into
	// []analysis.Analysis" 500. Guard the wire shape, not just the status code.
	var got []*analysis.Analysis
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("response body is not a bare array (VOIP-1199 regression): %v; body=%s", err, string(res.Data))
	}
	if len(got) != 1 || got[0].ActiveflowID != af {
		t.Fatalf("unexpected list payload: %s", string(res.Data))
	}
}

// Test_v1AnalysesGet_list_empty verifies that an empty result set still
// serializes as a bare JSON array ("[]"), not an object wrapper or "null".
// This pins the empty-list edge of the VOIP-1199 bare-array contract so the
// requesthandler's []analysis.Analysis unmarshal stays valid for zero rows.
func Test_v1AnalysesGet_list_empty(t *testing.T) {
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	m := &sock.Request{
		URI:    "/v1/analyses?customer_id=" + cust.String() + "&page_size=10",
		Method: sock.RequestMethodGet,
	}

	mockAnalysis.EXPECT().
		List(gomock.Any(), cust, "", uint64(10), gomock.Any()).
		Return([]*analysis.Analysis{}, nil)

	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var got []*analysis.Analysis
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("empty list body is not a bare array (VOIP-1199 regression): %v; body=%s", err, string(res.Data))
	}
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements: %s", len(got), string(res.Data))
	}
}

func Test_v1Analyses_routing_id_vs_collection(t *testing.T) {
	// the /<uuid> form must route to the ID handler, not the collection handler.
	h, mockAnalysis, ctrl := newAnalysisListenHandler(t)
	defer ctrl.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	m := &sock.Request{
		URI:    "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(),
		Method: sock.RequestMethodGet,
	}
	// expect Get (ID handler), never List.
	mockAnalysis.EXPECT().Get(gomock.Any(), cust, id).Return(&analysis.Analysis{}, nil)

	if _, err := h.processRequest(m); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// Test_v1Analyses_nilHandler_503 verifies that when the analysisHandler is nil
// (e.g. DATABASE_DSN unset, feature disabled), every analysis route returns
// 503 Service Unavailable instead of panicking (VOIP-1197 fail-safe).
func Test_v1Analyses_nilHandler_503(t *testing.T) {
	h := &listenHandler{analysisHandler: nil}

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	postBody, _ := json.Marshal(map[string]any{
		"customer_id":   cust.String(),
		"activeflow_id": af.String(),
	})

	tests := []struct {
		name string
		m    *sock.Request
	}{
		{
			name: "post",
			m:    &sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: postBody},
		},
		{
			name: "list",
			m:    &sock.Request{URI: "/v1/analyses?customer_id=" + cust.String(), Method: sock.RequestMethodGet},
		},
		{
			name: "id_get",
			m:    &sock.Request{URI: "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(), Method: sock.RequestMethodGet},
		},
		{
			name: "id_delete",
			m:    &sock.Request{URI: "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(), Method: sock.RequestMethodDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.processRequest(tt.m)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if res.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("expected 503, got %d", res.StatusCode)
			}
		})
	}
}
