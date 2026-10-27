package analysishandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

func Test_NewAnalysisHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	reqMock := requesthandler.NewMockRequestHandler(mc)
	dbMock := analysisdbhandler.NewMockAnalysisDBHandler(mc)
	evMock := eventhandler.NewMockEventHandler(mc)
	models := StageModels{Stage1: "test-model-1", Stage2: "test-model-2", Stage3: "test-model-3"}

	res := NewAnalysisHandler(reqMock, dbMock, evMock, models)

	h, ok := res.(*analysisHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *analysisHandler, got: %T", res)
	}

	if h.reqHandler != requesthandler.RequestHandler(reqMock) {
		t.Errorf("Wrong match. the request handler is not the injected one")
	}
	if h.dbHandler != analysisdbhandler.AnalysisDBHandler(dbMock) {
		t.Errorf("Wrong match. the database handler is not the injected one")
	}
	if h.eventHandler != eventhandler.EventHandler(evMock) {
		t.Errorf("Wrong match. the event handler is not the injected one")
	}
	if h.models != models {
		t.Errorf("Wrong match. expect: %+v, got: %+v", models, h.models)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. expect: a util handler, got: nil")
	}
	if h.sem == nil || cap(h.sem) != 8 {
		t.Errorf("Wrong match. expect a semaphore with capacity 8, got: %d", cap(h.sem))
	}
	if h.metricStarted != promAnalysisStarted || h.metricCompleted != promAnalysisCompleted ||
		h.metricDuration != promAnalysisDuration || h.metricEnrichment != promAnalysisEnrichment {
		t.Errorf("Wrong match. the metrics are not the package metrics")
	}
}
