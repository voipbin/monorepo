package analysishandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	amanalysis "monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

const runTestWait = 2 * time.Second

// dbUpdate is one captured AnalysisUpdateResult call.
type dbUpdate struct {
	id     uuid.UUID
	status analysis.Status
	result []byte
	model  string
	errStr string
}

// newRunTestHandler wraps newStartTestHandler: it exposes the event handler mock
// and swaps the process-wide metric collectors for fresh ones registered in a
// local registry, so values can be asserted without cross-test accumulation.
func newRunTestHandler(t *testing.T) (*analysisHandler, *requesthandler.MockRequestHandler, *analysisdbhandler.MockAnalysisDBHandler, *eventhandler.MockEventHandler, *prometheus.Registry, *gomock.Controller) {
	t.Helper()

	h, reqMock, dbMock, mc := newStartTestHandler(t)
	evMock := eventhandler.NewMockEventHandler(mc)
	h.eventHandler = evMock

	reg := prometheus.NewRegistry()
	h.metricStarted = prometheus.NewCounter(prometheus.CounterOpts{Name: "run_test_started_total", Help: "test"})
	h.metricCompleted = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "run_test_completed_total", Help: "test"}, []string{"status"})
	h.metricDuration = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "run_test_duration_seconds", Help: "test"})
	h.metricEnrichment = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "run_test_enrichment_total", Help: "test"}, []string{"reference_type", "outcome"})
	reg.MustRegister(h.metricStarted, h.metricCompleted, h.metricDuration, h.metricEnrichment)

	return h, reqMock, dbMock, evMock, reg, mc
}

// runTestMetric returns the gathered metric of a family whose labels match, and
// whether the family/series exists at all (a vec without children is absent).
func runTestMetric(t *testing.T, reg *prometheus.Registry, family string, labels map[string]string) (float64, uint64, bool) {
	t.Helper()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("could not gather metrics. err: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != family {
			continue
		}
		for _, m := range mf.GetMetric() {
			match := true
			for _, lp := range m.GetLabel() {
				if want, ok := labels[lp.GetName()]; ok && want != lp.GetValue() {
					match = false
				}
			}
			if match {
				return m.GetCounter().GetValue(), m.GetHistogram().GetSampleCount(), true
			}
		}
	}
	return 0, 0, false
}

// expectUpdateSignal expects exactly one AnalysisUpdateResult for the id and
// reports it on a buffered channel (so a failed test never blocks the goroutine).
func expectUpdateSignal(dbMock *analysisdbhandler.MockAnalysisDBHandler, id uuid.UUID) chan dbUpdate {
	ch := make(chan dbUpdate, 1)
	dbMock.EXPECT().AnalysisUpdateResult(gomock.Any(), id, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, gotID uuid.UUID, st analysis.Status, res []byte, model, errStr string) (int64, error) {
			ch <- dbUpdate{id: gotID, status: st, result: res, model: model, errStr: errStr}
			return 1, nil
		}).Times(1)
	return ch
}

func waitUpdate(t *testing.T, ch chan dbUpdate) dbUpdate {
	t.Helper()
	select {
	case u := <-ch:
		return u
	case <-time.After(runTestWait):
		t.Fatalf("timed out waiting for the AnalysisUpdateResult call (an unexpected mock call may have stopped the goroutine)")
		return dbUpdate{}
	}
}

// waitSemaphoreRelease relies on h.sem being a capacity-1 channel installed
// before kickoff: the send succeeds only after the goroutine's deferred release,
// which also orders its metric writes before the reads that follow.
func waitSemaphoreRelease(t *testing.T, h *analysisHandler) {
	t.Helper()
	select {
	case h.sem <- struct{}{}:
	case <-time.After(runTestWait):
		t.Fatalf("timed out waiting for the goroutine to release the semaphore slot")
	}
}

func Test_persistCompleted(t *testing.T) {
	id := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")

	tests := []struct {
		name string
		n    int64
		err  error
	}{
		{"persisted", 1, nil},
		{"row deleted or superseded", 0, nil},
		{"db error", 0, errors.New("db down")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, dbMock, _, _, mc := newRunTestHandler(t)
			defer mc.Finish()

			dbMock.EXPECT().AnalysisUpdateResult(gomock.Any(), id, analysis.StatusCompleted, []byte(`{"version":3}`), "model-x", "").
				DoAndReturn(func(ctx context.Context, _ uuid.UUID, _ analysis.Status, _ []byte, _, _ string) (int64, error) {
					if _, ok := ctx.Deadline(); !ok {
						t.Errorf("Wrong match. expect: a bounded write context, got: no deadline")
					}
					return tt.n, tt.err
				}).Times(1)

			h.persistCompleted(id, []byte(`{"version":3}`), "model-x")
		})
	}
}

func Test_persistFailed(t *testing.T) {
	id := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")

	tests := []struct {
		name string
		n    int64
		err  error
	}{
		{"persisted", 1, nil},
		{"row deleted or superseded", 0, nil},
		{"db error", 0, errors.New("db down")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, dbMock, _, _, mc := newRunTestHandler(t)
			defer mc.Finish()

			// the failed write carries no result bytes and no model.
			dbMock.EXPECT().AnalysisUpdateResult(gomock.Any(), id, analysis.StatusFailed, nil, "", "sanitized message").
				DoAndReturn(func(ctx context.Context, _ uuid.UUID, _ analysis.Status, _ []byte, _, _ string) (int64, error) {
					if _, ok := ctx.Deadline(); !ok {
						t.Errorf("Wrong match. expect: a bounded write context, got: no deadline")
					}
					return tt.n, tt.err
				}).Times(1)

			h.persistFailed(id, "sanitized message")
		})
	}
}

// kickoffFixture holds the ids shared by the kickoff tests.
type kickoffFixture struct {
	analysisID uuid.UUID
	customerID uuid.UUID
	afID       uuid.UUID
	af         *fmactiveflow.Activeflow
}

func newKickoffFixture() kickoffFixture {
	f := kickoffFixture{
		analysisID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000010"),
		customerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		afID:       uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
	}
	f.af = endedActiveflow(f.afID, f.customerID)
	return f
}

func Test_kickoff_success(t *testing.T) {
	h, reqMock, dbMock, evMock, reg, mc := newRunTestHandler(t)
	defer mc.Finish()
	f := newKickoffFixture()
	h.sem = make(chan struct{}, 1)

	evMock.EXPECT().AggregatedList(gomock.Any(), f.afID, "", 500).Return(&event.AggregatedEventListResponse{
		Result: []*event.Event{collectEvt(0, "call_created"), collectEvt(1, "call_hangup")},
	}, nil)
	reqMock.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   f.customerID,
		tmtranscribe.FieldActiveflowID: f.afID,
		tmtranscribe.FieldDeleted:      false,
	}).Return([]tmtranscribe.Transcribe{}, nil)

	var sawModel, sawSchemaName string
	reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), 120000).
		DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
			sawModel = req.Model
			sawSchemaName = req.SchemaName
			body := `{"overall_status":"warning","resources_used":[{"type":"llm-made-up","count":99}],` +
				`"interactions":[{"resource_type":"call","summary":"hangup after answer"}],"narrative":"call ended by hangup",` +
				`"issues":[{"severity":"warning","area":"call","summary":"hangup observed","evidence_index":[1]}]}`
			return &amanalysis.Response{Result: json.RawMessage(body), Model: "model-x", FinishReason: "stop"}, nil
		}).Times(1)
	signal := expectUpdateSignal(dbMock, f.analysisID)

	h.kickoff(f.analysisID, f.customerID, f.afID, f.af)

	got := waitUpdate(t, signal)
	waitSemaphoreRelease(t, h)

	if got.status != analysis.StatusCompleted || got.model != "model-x" || got.errStr != "" {
		t.Errorf("Wrong match. expect: completed/model-x/empty error, got: %s/%s/%s", got.status, got.model, got.errStr)
	}
	if sawModel != "m3" || sawSchemaName != "timeline_verdict" {
		t.Errorf("Wrong match. expect: the stage3 model and verdict schema, got: %s/%s", sawModel, sawSchemaName)
	}

	expectJSON := `{"version":3,"overall_status":"warning","input_reduced":false,` +
		`"session_context":{"reference_type":"","channel":"","multi_leg":false,"ai_handled":false,"human_involved":false},` +
		`"resources_used":[{"type":"call-manager","count":2}],` +
		`"interactions":[{"resource_type":"call","summary":"hangup after answer"}],"narrative":"call ended by hangup",` +
		`"issues":[{"severity":"warning","area":"call","summary":"hangup observed","evidence":[` +
		`{"evidence_index":1,"event_type":"call_hangup","timestamp":"2026-01-02T03:00:01.000000Z","resource_id":"r1"}]}]}`
	var expect, actual any
	if err := json.Unmarshal([]byte(expectJSON), &expect); err != nil {
		t.Fatalf("could not parse the expectation. err: %v", err)
	}
	if err := json.Unmarshal(got.result, &actual); err != nil {
		t.Fatalf("could not parse the persisted verdict. err: %v", err)
	}
	if !reflect.DeepEqual(expect, actual) {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expectJSON, string(got.result))
	}

	if v, _, ok := runTestMetric(t, reg, "run_test_started_total", nil); !ok || v != 1 {
		t.Errorf("Wrong match. expect: started 1, got: %v (found %v)", v, ok)
	}
	if v, _, ok := runTestMetric(t, reg, "run_test_completed_total", map[string]string{"status": "completed"}); !ok || v != 1 {
		t.Errorf("Wrong match. expect: completed 1, got: %v (found %v)", v, ok)
	}
	if _, _, ok := runTestMetric(t, reg, "run_test_completed_total", map[string]string{"status": "failed"}); ok {
		t.Errorf("Wrong match. expect: no failed series, got one")
	}
	if _, c, ok := runTestMetric(t, reg, "run_test_duration_seconds", nil); !ok || c != 1 {
		t.Errorf("Wrong match. expect: 1 duration sample, got: %d (found %v)", c, ok)
	}
}

func Test_kickoff_chain_error(t *testing.T) {
	h, _, dbMock, evMock, reg, mc := newRunTestHandler(t)
	defer mc.Finish()
	f := newKickoffFixture()
	h.sem = make(chan struct{}, 1)

	evMock.EXPECT().AggregatedList(gomock.Any(), f.afID, "", 500).Return(nil, errors.New("provider secret detail"))
	signal := expectUpdateSignal(dbMock, f.analysisID)

	h.kickoff(f.analysisID, f.customerID, f.afID, f.af)

	got := waitUpdate(t, signal)
	waitSemaphoreRelease(t, h)

	// only the sanitized message is persisted, never the underlying error text.
	if got.status != analysis.StatusFailed || got.errStr != "analysis failed to produce a result" || got.result != nil || got.model != "" {
		t.Errorf("Wrong match. expect: failed/sanitized message/no result/no model, got: %s/%q/%v/%q", got.status, got.errStr, got.result, got.model)
	}

	if v, _, ok := runTestMetric(t, reg, "run_test_started_total", nil); !ok || v != 1 {
		t.Errorf("Wrong match. expect: started 1, got: %v (found %v)", v, ok)
	}
	if v, _, ok := runTestMetric(t, reg, "run_test_completed_total", map[string]string{"status": "failed"}); !ok || v != 1 {
		t.Errorf("Wrong match. expect: failed 1, got: %v (found %v)", v, ok)
	}
	if v, _, ok := runTestMetric(t, reg, "run_test_completed_total", map[string]string{"status": "completed"}); ok && v != 0 {
		t.Errorf("Wrong match. expect: no completed count, got: %v", v)
	}
	if _, c, ok := runTestMetric(t, reg, "run_test_duration_seconds", nil); !ok || c != 1 {
		t.Errorf("Wrong match. expect: 1 duration sample, got: %d (found %v)", c, ok)
	}
}

func Test_kickoff_panic_recovered(t *testing.T) {
	h, _, dbMock, evMock, reg, mc := newRunTestHandler(t)
	defer mc.Finish()
	f := newKickoffFixture()
	h.sem = make(chan struct{}, 1)

	evMock.EXPECT().AggregatedList(gomock.Any(), f.afID, "", 500).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ string, _ int) (*event.AggregatedEventListResponse, error) {
			panic("synthetic failure")
		})
	signal := expectUpdateSignal(dbMock, f.analysisID)

	h.kickoff(f.analysisID, f.customerID, f.afID, f.af)

	got := waitUpdate(t, signal)
	waitSemaphoreRelease(t, h)

	if got.status != analysis.StatusFailed || got.errStr != "internal error during analysis" || got.result != nil {
		t.Errorf("Wrong match. expect: failed/internal error message/no result, got: %s/%q/%v", got.status, got.errStr, got.result)
	}

	if v, _, ok := runTestMetric(t, reg, "run_test_started_total", nil); !ok || v != 1 {
		t.Errorf("Wrong match. expect: started 1, got: %v (found %v)", v, ok)
	}
	// the recover path only persists; it records neither completed nor duration.
	if _, _, ok := runTestMetric(t, reg, "run_test_completed_total", nil); ok {
		t.Errorf("Wrong match. expect: no completed series, got one")
	}
	if _, c, _ := runTestMetric(t, reg, "run_test_duration_seconds", nil); c != 0 {
		t.Errorf("Wrong match. expect: 0 duration samples, got: %d", c)
	}
}
