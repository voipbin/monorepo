package listenhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysishandler"
)

func newAnalysisEdgeHandler(t *testing.T) (*listenHandler, *analysishandler.MockAnalysisHandler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	mockAnalysis := analysishandler.NewMockAnalysisHandler(mc)
	return &listenHandler{analysisHandler: mockAnalysis}, mockAnalysis, mc
}

func Test_analysisErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectStatus int
	}{
		{"reanalyze cooldown", analysishandler.ErrReanalyzeCooldown, http.StatusTooManyRequests},
		{"concurrency limit", analysishandler.ErrConcurrencyLimit, http.StatusTooManyRequests},
		{"activeflow not ended", analysishandler.ErrActiveflowNotEnded, http.StatusConflict},
		{"not found", analysishandler.ErrNotFound, http.StatusNotFound},
		{"wrapped sentinel", errors.Join(errors.New("outer"), analysishandler.ErrConcurrencyLimit), http.StatusTooManyRequests},
		{"unclassified", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := analysisErrorResponse(tt.err)
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_v1AnalysesPost_invalid_request(t *testing.T) {
	cust := "11111111-1111-1111-1111-111111111111"
	af := "22222222-2222-2222-2222-222222222222"
	nilID := "00000000-0000-0000-0000-000000000000"

	tests := []struct {
		name string
		data []byte
	}{
		{"malformed json", []byte(`{"customer_id":`)},
		// uuid.Nil is the invalid input under test, not an expected value.
		{"nil customer id", []byte(`{"customer_id":"` + nilID + `","activeflow_id":"` + af + `"}`)},
		{"nil activeflow id", []byte(`{"customer_id":"` + cust + `","activeflow_id":"` + nilID + `"}`)},
		{"missing ids", []byte(`{}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no EXPECT: the handler must not be reached for an invalid request.
			h, _, mc := newAnalysisEdgeHandler(t)
			defer mc.Finish()

			m := &sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: tt.data}
			res, err := h.processRequest(m)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

func Test_v1AnalysesPost_handler_error(t *testing.T) {
	h, mockAnalysis, mc := newAnalysisEdgeHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	body := []byte(`{"customer_id":"11111111-1111-1111-1111-111111111111","activeflow_id":"22222222-2222-2222-2222-222222222222","reanalyze":true}`)
	mockAnalysis.EXPECT().Start(gomock.Any(), cust, af, true).Return(nil, analysishandler.ErrConcurrencyLimit)

	res, err := h.processRequest(&sock.Request{URI: "/v1/analyses", Method: sock.RequestMethodPost, Data: body})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusTooManyRequests, res.StatusCode)
	}
}

func Test_v1AnalysesGet_invalid_request(t *testing.T) {
	cust := "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		name string
		uri  string
		data []byte
	}{
		{"missing customer id", "/v1/analyses", nil},
		{"customer id not a uuid", "/v1/analyses?customer_id=not-a-uuid", nil},
		{"unparsable query string", "/v1/analyses?customer_id=%zz", nil},
		{"malformed filter body", "/v1/analyses?customer_id=" + cust, []byte(`{"status":`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, mc := newAnalysisEdgeHandler(t)
			defer mc.Finish()

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

func Test_v1AnalysesGet_filters(t *testing.T) {
	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	af := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	h, mockAnalysis, mc := newAnalysisEdgeHandler(t)
	defer mc.Finish()

	// the body customer_id must be dropped (the query param is the authority);
	// *_id values are converted to uuid.UUID and other values pass through.
	body, _ := json.Marshal(map[string]any{
		"customer_id":   other.String(),
		"activeflow_id": af.String(),
		"id":            id.String(),
		"status":        "completed",
	})
	expectFilters := map[analysis.Field]any{
		analysis.FieldActiveflowID: af,
		analysis.FieldID:           id,
		analysis.FieldStatus:       "completed",
	}
	row := &analysis.Analysis{ActiveflowID: af}
	mockAnalysis.EXPECT().List(gomock.Any(), cust, "2026-01-01T00:00:00.000000Z", uint64(25), expectFilters).Return([]*analysis.Analysis{row}, nil)

	m := &sock.Request{
		URI:    "/v1/analyses?customer_id=" + cust.String() + "&page_token=2026-01-01T00:00:00.000000Z&page_size=25",
		Method: sock.RequestMethodGet,
		Data:   body,
	}
	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
	}

	if res.DataType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %s", res.DataType)
	}

	var got []*analysis.Analysis
	if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: array body, got: %v", errUnmarshal)
	}
	if len(got) != 1 || got[0].ActiveflowID != af {
		t.Errorf("Wrong match. expect: 1 row for %s, got: %s", af, string(res.Data))
	}
}

func Test_v1AnalysesGet_handler_error(t *testing.T) {
	h, mockAnalysis, mc := newAnalysisEdgeHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	mockAnalysis.EXPECT().List(gomock.Any(), cust, "", uint64(0), map[analysis.Field]any{}).Return(nil, errors.New("boom"))

	res, err := h.processRequest(&sock.Request{URI: "/v1/analyses?customer_id=" + cust.String(), Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func Test_v1AnalysesID_invalid_request(t *testing.T) {
	cust := "11111111-1111-1111-1111-111111111111"
	// the route pattern needs a well-formed uuid, so the nil uuid is the way to
	// reach the invalid id check through the router.
	nilID := "00000000-0000-0000-0000-000000000000"
	id := "33333333-3333-3333-3333-333333333333"

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"get nil id", sock.RequestMethodGet, "/v1/analyses/" + nilID + "?customer_id=" + cust},
		{"get missing customer id", sock.RequestMethodGet, "/v1/analyses/" + id},
		{"delete nil id", sock.RequestMethodDelete, "/v1/analyses/" + nilID + "?customer_id=" + cust},
		{"delete missing customer id", sock.RequestMethodDelete, "/v1/analyses/" + id},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, mc := newAnalysisEdgeHandler(t)
			defer mc.Finish()

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

func Test_v1AnalysesIDDelete_handler_error(t *testing.T) {
	h, mockAnalysis, mc := newAnalysisEdgeHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	mockAnalysis.EXPECT().Delete(gomock.Any(), cust, id).Return(nil, analysishandler.ErrNotFound)

	m := &sock.Request{URI: "/v1/analyses/" + id.String() + "?customer_id=" + cust.String(), Method: sock.RequestMethodDelete}
	res, err := h.processRequest(m)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, res.StatusCode)
	}
}

func Test_analysisIDFromURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string

		expectID uuid.UUID
		expectOK bool
	}{
		{"plain", "/v1/analyses/33333333-3333-3333-3333-333333333333", uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"), true},
		{"with query", "/v1/analyses/33333333-3333-3333-3333-333333333333?customer_id=x", uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"), true},
		{"collection with trailing slash has no id", "/v1/analyses/", uuid.Nil, false},
		{"collection path has no id", "/v1/analyses", uuid.Nil, false},
		{"id segment is not a uuid", "/v1/analyses/abc", uuid.Nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := analysisIDFromURI(tt.uri)
			if ok != tt.expectOK || id != tt.expectID {
				t.Errorf("Wrong match. expect: %s %v, got: %s %v", tt.expectID, tt.expectOK, id, ok)
			}
		})
	}
}

func Test_normalizeFilterValue(t *testing.T) {
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name  string
		field analysis.Field
		value any

		expect any
	}{
		{"id string to uuid", analysis.FieldID, id.String(), id},
		{"activeflow id string to uuid", analysis.FieldActiveflowID, id.String(), id},
		{"customer id string to uuid", analysis.FieldCustomerID, id.String(), id},
		{"invalid id string becomes nil uuid", analysis.FieldID, "not-a-uuid", uuid.Nil},
		{"non string id passes through", analysis.FieldID, float64(7), float64(7)},
		{"status passes through", analysis.FieldStatus, "failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFilterValue(tt.field, tt.value)
			if got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_parsePageSize(t *testing.T) {
	tests := []struct {
		in     string
		expect uint64
	}{
		{"", 0},
		{"10", 10},
		{"9", 9},
		{"19", 19},
		{"90", 90},
		{"007", 7},
		{"-5", 0},
		{"1x", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run("input_"+tt.in, func(t *testing.T) {
			if got := parsePageSize(tt.in); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}
