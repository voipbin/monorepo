package analysishandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

func newCollectTestHandler(t *testing.T) (*analysisHandler, *requesthandler.MockRequestHandler, *eventhandler.MockEventHandler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	reqMock := requesthandler.NewMockRequestHandler(mc)
	evMock := eventhandler.NewMockEventHandler(mc)

	h := &analysisHandler{
		utilHandler:  utilhandler.NewUtilHandler(),
		reqHandler:   reqMock,
		dbHandler:    analysisdbhandler.NewMockAnalysisDBHandler(mc),
		eventHandler: evMock,
		models:       StageModels{Stage1: "m1", Stage2: "m2", Stage3: "m3"},
		sem:          make(chan struct{}, analysisMaxConcurrentJobs),
	}
	return h, reqMock, evMock, mc
}

var collectTestBase = time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

// collectEvt builds a distinct event: the timestamp and the payload id both depend on i.
func collectEvt(i int, eventType string) *event.Event {
	return &event.Event{
		Timestamp: collectTestBase.Add(time.Duration(i) * time.Second),
		EventType: eventType,
		Publisher: commonoutline.ServiceName("call-manager"),
		Data:      []byte(fmt.Sprintf(`{"id":"r%d"}`, i)),
	}
}

func collectEvts(from, to int, eventType string) []*event.Event {
	res := []*event.Event{}
	for i := from; i < to; i++ {
		res = append(res, collectEvt(i, eventType))
	}
	return res
}

func Test_collectEvents_list_error(t *testing.T) {
	h, _, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	boom := errors.New("backend failure")

	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(nil, boom)

	got, capped, err := h.collectEvents(ctx, afID)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "collectEvents: could not list events") {
		t.Errorf("Wrong match. expect: wrapped backend failure, got: %v", err)
	}
	if got != nil || capped {
		t.Errorf("Wrong match. expect: nil and not capped, got: %v %v", got, capped)
	}
}

func Test_collectEvents_empty_pages(t *testing.T) {
	tests := []struct {
		name string
		resp *event.AggregatedEventListResponse
	}{
		{"nil response", nil},
		{"empty result with a next token still stops", &event.AggregatedEventListResponse{NextPageToken: "tok-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, evMock, mc := newCollectTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

			evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(tt.resp, nil).Times(1)

			got, capped, err := h.collectEvents(ctx, afID)
			if err != nil || capped || len(got) != 0 {
				t.Errorf("Wrong match. expect: no events, got: %d events capped=%v err=%v", len(got), capped, err)
			}
		})
	}
}

func Test_collectEvents_pagination_and_dedup(t *testing.T) {
	h, _, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	e0 := collectEvt(0, "call_created")
	e1 := collectEvt(1, "call_created")
	e2 := collectEvt(2, "call_created")

	// page 1 holds a duplicate of e0 inside the page; page 2 repeats e1.
	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{
		Result:        []*event.Event{e0, collectEvt(0, "call_created"), e1},
		NextPageToken: "tok-1",
	}, nil)
	evMock.EXPECT().AggregatedList(ctx, afID, "tok-1", 500).Return(&event.AggregatedEventListResponse{
		Result: []*event.Event{collectEvt(1, "call_created"), e2},
	}, nil)

	got, capped, err := h.collectEvents(ctx, afID)
	if err != nil || capped {
		t.Fatalf("Wrong match. expect: ok and not capped, got: capped=%v err=%v", capped, err)
	}

	var raws []*event.Event
	for _, ce := range got {
		raws = append(raws, ce.raw)
	}
	if !reflect.DeepEqual(raws, []*event.Event{e0, e1, e2}) {
		t.Errorf("Wrong match. expect: e0,e1,e2 in order, got %d events", len(raws))
	}
}

func Test_collectEvents_dedup_key_fields(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	base := func() *event.Event {
		return &event.Event{Timestamp: ts, EventType: "call_created", Data: []byte(`{"id":"r1"}`)}
	}

	tests := []struct {
		name   string
		second func() *event.Event

		expectCount int
	}{
		{"identical rows are deduplicated", base, 1},
		{"data only", func() *event.Event { e := base(); e.Data = []byte(`{"id":"r1","k":2}`); return e }, 2},
		{"type only", func() *event.Event { e := base(); e.EventType = "call_updated"; return e }, 2},
		{"one microsecond apart", func() *event.Event { e := base(); e.Timestamp = ts.Add(time.Microsecond); return e }, 2},
		{"below a microsecond is the same row", func() *event.Event { e := base(); e.Timestamp = ts.Add(time.Nanosecond); return e }, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, evMock, mc := newCollectTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

			evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{
				Result: []*event.Event{base(), tt.second()},
			}, nil)

			got, _, err := h.collectEvents(ctx, afID)
			if err != nil || len(got) != tt.expectCount {
				t.Errorf("Wrong match. expect: %d events, got: %d err=%v", tt.expectCount, len(got), err)
			}
		})
	}
}

func Test_collectEvents_event_cap(t *testing.T) {
	tests := []struct {
		name  string
		count int

		expectLen    int
		expectCapped bool
	}{
		{"4999 events are not capped", 4999, 4999, false},
		{"exactly 5000 events are capped", 5000, 5000, true},
		{"more than 5000 stops at 5000", 5001, 5000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, evMock, mc := newCollectTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

			// a next token is offered, but the 5000th event ends the loop first when it is reached.
			evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{
				Result: collectEvts(0, tt.count, "call_created"),
			}, nil).Times(1)

			got, capped, err := h.collectEvents(ctx, afID)
			if err != nil || len(got) != tt.expectLen || capped != tt.expectCapped {
				t.Errorf("Wrong match. expect: %d capped=%v, got: %d capped=%v err=%v", tt.expectLen, tt.expectCapped, len(got), capped, err)
			}
		})
	}
}

func Test_collectEvents_page_limit(t *testing.T) {
	h, _, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	// one new event per page and always another page: only the 100 page limit ends the loop.
	for i := 0; i < 100; i++ {
		token := ""
		if i > 0 {
			token = fmt.Sprintf("tok-%d", i)
		}
		evMock.EXPECT().AggregatedList(ctx, afID, token, 500).Return(&event.AggregatedEventListResponse{
			Result:        []*event.Event{collectEvt(i, "call_created")},
			NextPageToken: fmt.Sprintf("tok-%d", i+1),
		}, nil).Times(1)
	}

	got, capped, err := h.collectEvents(ctx, afID)
	if err != nil || capped || len(got) != 100 {
		t.Errorf("Wrong match. expect: 100 events not capped, got: %d capped=%v err=%v", len(got), capped, err)
	}
}

func Test_collectInput_list_error(t *testing.T) {
	h, _, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	custID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	boom := errors.New("backend failure")

	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(nil, boom)

	got, err := h.collectInput(ctx, custID, afID)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "collectInput: could not collect events") {
		t.Errorf("Wrong match. expect: wrapped backend failure, got: %v", err)
	}
	if got != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", got)
	}
}

func Test_collectInput_freeze_and_signals(t *testing.T) {
	h, reqMock, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	custID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	trID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333331")

	flowEvt := collectEvt(2, "activeflow_created")
	flowEvt.Publisher = commonoutline.ServiceName("flow-manager")
	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{
		Result: []*event.Event{collectEvt(0, "call_created"), collectEvt(1, "call_hangup"), flowEvt},
	}, nil)

	reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   custID,
		tmtranscribe.FieldActiveflowID: afID,
		tmtranscribe.FieldDeleted:      false,
	}).Return([]tmtranscribe.Transcribe{{Identity: commonidentity.Identity{ID: trID}}}, nil)
	reqMock.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
		tmtranscript.FieldTranscribeID: trID,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{{Direction: tmtranscript.DirectionIn, Message: "hi"}}, nil)

	got, err := h.collectInput(ctx, custID, afID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(got.events) != 3 || len(got.allEvents) != 3 {
		t.Fatalf("Wrong match. expect: 3 events and 3 allEvents, got: %d and %d", len(got.events), len(got.allEvents))
	}
	for i, ce := range got.events {
		if ce.Index != i {
			t.Errorf("Wrong match. expect index %d, got: %d", i, ce.Index)
		}
	}
	if len(got.errorSignals) != 1 || got.errorSignals[0].EventType != "call_hangup" || got.errorSignals[0].Index != 1 {
		t.Errorf("Wrong match. expect: only the call_hangup signal at index 1, got: %v", got.errorSignals)
	}
	expectInventory := []resourceCount{{Type: "call-manager", Count: 2}, {Type: "flow-manager", Count: 1}}
	if !reflect.DeepEqual(got.inventory, expectInventory) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectInventory, got.inventory)
	}
	if !reflect.DeepEqual(got.transcripts, []string{"[in] hi"}) {
		t.Errorf("Wrong match. expect: [[in] hi], got: %v", got.transcripts)
	}
	if got.inputReduced {
		t.Errorf("Wrong match. expect: not reduced")
	}
}

func Test_collectInput_reduced_by_size_only(t *testing.T) {
	h, reqMock, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	custID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	// 400 non-error events with a 200 character message each exceed the 96 KiB target.
	evts := []*event.Event{}
	for i := 0; i < 400; i++ {
		e := collectEvt(i, "call_updated")
		e.Data = []byte(fmt.Sprintf(`{"message":"%s","id":"r%d"}`, strings.Repeat("m", 200), i))
		evts = append(evts, e)
	}
	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{Result: evts}, nil)
	reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), gomock.Any()).Return(nil, nil)

	got, err := h.collectInput(ctx, custID, afID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !got.inputReduced {
		t.Errorf("Wrong match. expect: reduced by size")
	}
	if len(got.allEvents) != 400 {
		t.Errorf("Wrong match. expect: allEvents keeps all 400, got: %d", len(got.allEvents))
	}
	if len(got.events) == 0 || len(got.events) >= 400 {
		t.Fatalf("Wrong match. expect: some but not all events kept, got: %d", len(got.events))
	}
	for i, ce := range got.events {
		if ce.Index != i {
			t.Fatalf("Wrong match. expect contiguous index %d, got: %d", i, ce.Index)
		}
	}
}

func Test_collectInput_reduced_by_cap_only(t *testing.T) {
	h, reqMock, evMock, mc := newCollectTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()
	custID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	// error events always survive size reduction, so only the 5000 event cap marks the input reduced.
	evMock.EXPECT().AggregatedList(ctx, afID, "", 500).Return(&event.AggregatedEventListResponse{
		Result: collectEvts(0, 5001, "call_error"),
	}, nil)
	reqMock.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), gomock.Any()).Return(nil, nil)

	got, err := h.collectInput(ctx, custID, afID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !got.inputReduced || len(got.events) != 5000 || len(got.errorSignals) != 5000 {
		t.Errorf("Wrong match. expect: reduced with 5000 events and signals, got: reduced=%v events=%d signals=%d", got.inputReduced, len(got.events), len(got.errorSignals))
	}
}

func Test_dedupKey(t *testing.T) {
	e := &event.Event{
		Timestamp: time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC),
		EventType: "call_created",
		Data:      []byte(`{"id":"r1"}`),
	}

	if got, expect := dedupKey(e), "2026-01-02T03:04:05.123456Z|call_created|r1|920e5591b811bf1b"; got != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
	}
}

func Test_buildInventory(t *testing.T) {
	events := []*canonicalEvent{
		{Publisher: "zeta"},
		{Publisher: "alpha"},
		{Publisher: ""},
		{Publisher: "alpha"},
	}

	expect := []resourceCount{{Type: "alpha", Count: 2}, {Type: "unknown", Count: 1}, {Type: "zeta", Count: 1}}
	if got := buildInventory(events); !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
	}

	if got := buildInventory(nil); len(got) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", got)
	}
}

func Test_isErrorEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expect    bool
	}{
		{"call_error", true},
		{"call_failed", true},
		{"call_hangup", true},
		{"call_timeout", true},
		{"CALL_HANGUP", true},
		{"call_created", false},
		{"error", false},
		{"hangup", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := isErrorEvent(tt.eventType); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
