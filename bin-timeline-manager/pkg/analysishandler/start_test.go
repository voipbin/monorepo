package analysishandler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

func newStartTestHandler(t *testing.T) (*analysisHandler, *requesthandler.MockRequestHandler, *analysisdbhandler.MockAnalysisDBHandler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	reqMock := requesthandler.NewMockRequestHandler(mc)
	dbMock := analysisdbhandler.NewMockAnalysisDBHandler(mc)
	evMock := eventhandler.NewMockEventHandler(mc)

	h := &analysisHandler{
		utilHandler:  utilhandler.NewUtilHandler(),
		reqHandler:   reqMock,
		dbHandler:    dbMock,
		eventHandler: evMock,
		models:       StageModels{Stage1: "m1", Stage2: "m2", Stage3: "m3"},
		sem:          make(chan struct{}, analysisMaxConcurrentJobs),
		metricStarted:   promAnalysisStarted,
		metricCompleted: promAnalysisCompleted,
		metricDuration:  promAnalysisDuration,
	}
	return h, reqMock, dbMock, mc
}

func endedActiveflow(id, customerID uuid.UUID) *fmactiveflow.Activeflow {
	af := &fmactiveflow.Activeflow{}
	af.Identity = commonidentity.Identity{ID: id, CustomerID: customerID}
	af.Status = fmactiveflow.StatusEnded
	return af
}

func Test_Start_foreign_activeflow_masked(t *testing.T) {
	h, reqMock, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")

	// activeflow owned by someone else -> masked not-found, no DB touch.
	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, other), nil)

	_, err := h.Start(context.Background(), cust, afID, false)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func Test_Start_not_ended(t *testing.T) {
	h, reqMock, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	af := &fmactiveflow.Activeflow{}
	af.Identity = commonidentity.Identity{ID: afID, CustomerID: cust}
	af.Status = fmactiveflow.StatusRunning
	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(af, nil)

	_, err := h.Start(context.Background(), cust, afID, false)
	if err != ErrActiveflowNotEnded {
		t.Fatalf("expected ErrActiveflowNotEnded, got %v", err)
	}
}

func Test_Start_existing_completed_idempotent(t *testing.T) {
	h, reqMock, dbMock, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)

	existing := &analysis.Analysis{Status: analysis.StatusCompleted}
	existing.Identity = commonidentity.Identity{CustomerID: cust}
	existing.ActiveflowID = afID
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(existing, nil)

	res, err := h.Start(context.Background(), cust, afID, false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != analysis.StatusCompleted {
		t.Fatalf("expected idempotent completed return, got %v", res.Status)
	}
}

func Test_Start_reanalyze_cooldown(t *testing.T) {
	h, reqMock, dbMock, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)

	recent := time.Now().Add(-10 * time.Second) // within 1-min cooldown
	existing := &analysis.Analysis{Status: analysis.StatusCompleted, TMUpdate: &recent}
	existing.Identity = commonidentity.Identity{CustomerID: cust}
	existing.ActiveflowID = afID
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(existing, nil)

	_, err := h.Start(context.Background(), cust, afID, true)
	if err != ErrReanalyzeCooldown {
		t.Fatalf("expected ErrReanalyzeCooldown, got %v", err)
	}
}

func Test_Start_existing_progressing_no_double_spend(t *testing.T) {
	h, reqMock, dbMock, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)

	existing := &analysis.Analysis{Status: analysis.StatusProgressing}
	existing.Identity = commonidentity.Identity{CustomerID: cust}
	existing.ActiveflowID = afID
	// reanalyze=true while progressing must still be a no-op return (no reset call).
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(existing, nil)

	res, err := h.Start(context.Background(), cust, afID, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != analysis.StatusProgressing {
		t.Fatalf("expected in-flight progressing return, got %v", res.Status)
	}
}

func Test_Start_create_dup_returns_inflight(t *testing.T) {
	h, reqMock, dbMock, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)
	// no existing row.
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, analysisdbhandler.ErrNotFound)
	// under the concurrency cap.
	dbMock.EXPECT().AnalysisCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	// create loses the race -> duplicate.
	dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).Return(analysisdbhandler.ErrDuplicate)
	// re-read returns the concurrent in-flight row.
	inflight := &analysis.Analysis{Status: analysis.StatusProgressing}
	inflight.Identity = commonidentity.Identity{CustomerID: cust}
	inflight.ActiveflowID = afID
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(inflight, nil)

	res, err := h.Start(context.Background(), cust, afID, false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != analysis.StatusProgressing {
		t.Fatalf("expected in-flight progressing return, got %v", res.Status)
	}
}

func Test_Start_concurrency_cap(t *testing.T) {
	h, reqMock, dbMock, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)
	// no existing row -> new analysis path.
	dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, analysisdbhandler.ErrNotFound)
	// at the cap -> reject, never create.
	dbMock.EXPECT().AnalysisCountProgressing(gomock.Any(), cust).Return(int64(analysisMaxProgressingPerCustomer), nil)

	_, err := h.Start(context.Background(), cust, afID, false)
	if err != ErrConcurrencyLimit {
		t.Fatalf("expected ErrConcurrencyLimit, got %v", err)
	}
}
