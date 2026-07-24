package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	call "monorepo/bin-call-manager/models/call"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func Test_processEventRun(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "normal",
			event: &sock.Event{
				Type:      "call_created",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"test-id"}`),
			},
		},
		{
			name: "empty fields",
			event: &sock.Event{
				Type:      "",
				Publisher: "",
				DataType:  "",
				Data:      json.RawMessage(`{}`),
			},
		},
		{
			name: "nil data",
			event: &sock.Event{
				Type:      "agent_updated",
				Publisher: "agent-manager",
				DataType:  "application/json",
				Data:      nil,
			},
		},
		{
			name: "large data payload",
			event: &sock.Event{
				Type:      "flow_executed",
				Publisher: "flow-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"key":"` + string(make([]byte, 4096)) + `"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &subscribeHandler{
				eventCh: make(chan *sock.Event, eventChBuffer),
			}

			if err := h.processEventRun(tt.event); err != nil {
				t.Errorf("processEventRun() returned error: %v", err)
			}

			select {
			case got := <-h.eventCh:
				if got.Type != tt.event.Type {
					t.Errorf("event type = %s, want %s", got.Type, tt.event.Type)
				}
				if got.Publisher != tt.event.Publisher {
					t.Errorf("event publisher = %s, want %s", got.Publisher, tt.event.Publisher)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("event was not pushed to channel")
			}
		})
	}
}

func Test_processEventRun_multipleSequential(t *testing.T) {
	h := &subscribeHandler{
		eventCh: make(chan *sock.Event, eventChBuffer),
	}

	events := []*sock.Event{
		{Type: "call_created", Publisher: "call-manager"},
		{Type: "flow_updated", Publisher: "flow-manager"},
		{Type: "agent_deleted", Publisher: "agent-manager"},
	}

	for _, e := range events {
		if err := h.processEventRun(e); err != nil {
			t.Fatalf("processEventRun() returned error: %v", err)
		}
	}

	if len(h.eventCh) != len(events) {
		t.Errorf("channel length = %d, want %d", len(h.eventCh), len(events))
	}

	// Verify ordering is preserved (FIFO)
	for i, want := range events {
		got := <-h.eventCh
		if got.Type != want.Type {
			t.Errorf("event[%d] type = %s, want %s", i, got.Type, want.Type)
		}
	}
}

func Test_processEventRun_channelFull(t *testing.T) {
	h := &subscribeHandler{
		eventCh: make(chan *sock.Event, 1), // buffer of 1
	}

	// Fill the channel
	h.eventCh <- &sock.Event{Type: "first"}

	// This should not block — it drops the event
	err := h.processEventRun(&sock.Event{Type: "second"})
	if err != nil {
		t.Errorf("processEventRun() returned error: %v", err)
	}

	// Channel should still have only the first event
	got := <-h.eventCh
	if got.Type != "first" {
		t.Errorf("expected first event, got %s", got.Type)
	}

	// Channel should be empty now
	select {
	case extra := <-h.eventCh:
		t.Errorf("channel should be empty, got event: %s", extra.Type)
	default:
		// expected
	}
}

func Test_flushBatch(t *testing.T) {
	tests := []struct {
		name    string
		entries []eventEntry
	}{
		{
			name: "single event",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "call_created",
						Publisher: "call-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-id"}`),
					},
					receivedAt: time.Now(),
				},
			},
		},
		{
			name: "multiple events from same publisher",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "call_created",
						Publisher: "call-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-1"}`),
					},
					receivedAt: time.Now(),
				},
				{
					event: &sock.Event{
						Type:      "call_hangup",
						Publisher: "call-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-2"}`),
					},
					receivedAt: time.Now(),
				},
			},
		},
		{
			name: "multiple events from different publishers",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "call_created",
						Publisher: "call-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-1"}`),
					},
					receivedAt: time.Now(),
				},
				{
					event: &sock.Event{
						Type:      "flow_updated",
						Publisher: "flow-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-2"}`),
					},
					receivedAt: time.Now(),
				},
				{
					event: &sock.Event{
						Type:      "agent_deleted",
						Publisher: "agent-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-3"}`),
					},
					receivedAt: time.Now(),
				},
			},
		},
		{
			name: "event with empty data",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "billing_updated",
						Publisher: "billing-manager",
						DataType:  "",
						Data:      nil,
					},
					receivedAt: time.Now(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &subscribeHandler{
				dbHandler: mockDB,
			}

			mockDB.EXPECT().EventBatchInsert(
				gomock.Any(),
				gomock.Len(len(tt.entries)),
			).Return(nil)

			h.flushBatch(tt.entries)
		})
	}
}

func Test_flushBatch_fieldMapping(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
	}

	ts := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	entries := []eventEntry{
		{
			event: &sock.Event{
				Type:      "call_created",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"abc-123"}`),
			},
			receivedAt: ts,
		},
	}

	// Verify exact field mapping from sock.Event -> EventRow
	mockDB.EXPECT().EventBatchInsert(
		gomock.Any(),
		gomock.Eq([]dbhandler.EventRow{
			{
				Timestamp: ts,
				EventType: "call_created",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      `{"id":"abc-123"}`,
			},
		}),
	).Return(nil)

	h.flushBatch(entries)
}

func Test_flushBatch_fullBatchSize(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
	}

	entries := make([]eventEntry, batchSize)
	for i := range entries {
		entries[i] = eventEntry{
			event: &sock.Event{
				Type:      fmt.Sprintf("event_%d", i),
				Publisher: "test-publisher",
				DataType:  "application/json",
				Data:      json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)),
			},
			receivedAt: time.Now(),
		}
	}

	mockDB.EXPECT().EventBatchInsert(
		gomock.Any(),
		gomock.Len(batchSize),
	).Return(nil)

	h.flushBatch(entries)
}

// Test_flushBatch_EventBatchInsertFailure_SkipsPeerEventProjection documents
// and locks in the contract implicit in Test_flushBatch_error: when the
// primary EventBatchInsert fails, flushBatch returns before reaching
// buildPeerEventRows/PeerEventBatchInsert at all — a transient ClickHouse
// failure on the primary `events` write also means that batch's peer_events
// projection is silently skipped for this flush cycle (no separate retry).
// This is intentional (events is the audit-log source of truth; peer_events
// is a secondary projection derived from the same batch, not an independent
// write), flagged in PR #1135 Round 1 review as worth locking in explicitly.
// The mock's strict unexpected-call enforcement (no EXPECT() for
// PeerEventBatchInsert) means this test fails loudly if that ordering ever
// regresses.
func Test_flushBatch_EventBatchInsertFailure_SkipsPeerEventProjection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	// call_created is peer_events-eligible per eligiblePeerEvents, so if the
	// early-return-on-error guard were ever removed, buildPeerEventRows would
	// produce a row and PeerEventBatchInsert would be called -- which this
	// mock does NOT expect, so gomock would fail the test.
	entries := []eventEntry{
		{
			event: &sock.Event{
				Type:      call.EventTypeCallCreated,
				Publisher: string(commonoutline.ServiceNameCallManager),
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"` + uuid.Must(uuid.NewV4()).String() + `"}`),
			},
			receivedAt: time.Now(),
		},
	}

	mockDB.EXPECT().EventBatchInsert(gomock.Any(), gomock.Any()).Return(fmt.Errorf("connection lost"))

	h := &subscribeHandler{dbHandler: mockDB}
	h.flushBatch(entries)
}

func Test_flushBatch_error(t *testing.T) {
	tests := []struct {
		name      string
		entries   []eventEntry
		insertErr error
	}{
		{
			name: "EventBatchInsert returns error",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "flow_updated",
						Publisher: "flow-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-id"}`),
					},
					receivedAt: time.Now(),
				},
			},
			insertErr: fmt.Errorf("connection lost"),
		},
		{
			name: "error with multiple events",
			entries: []eventEntry{
				{
					event: &sock.Event{
						Type:      "call_created",
						Publisher: "call-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-1"}`),
					},
					receivedAt: time.Now(),
				},
				{
					event: &sock.Event{
						Type:      "flow_updated",
						Publisher: "flow-manager",
						DataType:  "application/json",
						Data:      json.RawMessage(`{"id":"test-2"}`),
					},
					receivedAt: time.Now(),
				},
			},
			insertErr: fmt.Errorf("clickhouse timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &subscribeHandler{
				dbHandler: mockDB,
			}

			mockDB.EXPECT().EventBatchInsert(
				gomock.Any(),
				gomock.Len(len(tt.entries)),
			).Return(tt.insertErr)

			// Should not panic on error
			h.flushBatch(tt.entries)
		})
	}
}

func Test_flushWorker_flushOnBatchSize(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// Expect exactly one batch insert with batchSize events
	done := make(chan struct{})
	mockDB.EXPECT().EventBatchInsert(
		gomock.Any(),
		gomock.Len(batchSize),
	).DoAndReturn(func(_ interface{}, rows []dbhandler.EventRow) error {
		close(done)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.flushWorker(ctx)

	// Push exactly batchSize events
	for i := 0; i < batchSize; i++ {
		h.eventCh <- &sock.Event{
			Type:      fmt.Sprintf("event_%d", i),
			Publisher: "test-publisher",
			DataType:  "application/json",
			Data:      json.RawMessage(`{}`),
		}
	}

	select {
	case <-done:
		// Batch was flushed by batchSize threshold
	case <-time.After(500 * time.Millisecond):
		t.Error("flushWorker did not flush on batchSize threshold")
	}
}

func Test_flushWorker_flushOnTimer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// Expect one batch insert with fewer than batchSize events (timer-triggered)
	done := make(chan struct{})
	mockDB.EXPECT().EventBatchInsert(
		gomock.Any(),
		gomock.Len(3),
	).DoAndReturn(func(_ interface{}, rows []dbhandler.EventRow) error {
		close(done)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.flushWorker(ctx)

	// Push fewer than batchSize events
	for i := 0; i < 3; i++ {
		h.eventCh <- &sock.Event{
			Type:      fmt.Sprintf("event_%d", i),
			Publisher: "test-publisher",
			DataType:  "application/json",
			Data:      json.RawMessage(`{}`),
		}
	}

	select {
	case <-done:
		// Batch was flushed by timer
	case <-time.After(3 * time.Second):
		t.Error("flushWorker did not flush on timer interval")
	}
}

func Test_flushWorker_noFlushOnEmptyBuffer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// EventBatchInsert should NOT be called when there are no events.
	// gomock will fail if it gets an unexpected call.
	// Use context cancellation to stop the worker cleanly instead of time.Sleep.
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		h.flushWorker(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Worker exited after context timeout — no unexpected calls
	case <-time.After(5 * time.Second):
		t.Error("flushWorker did not exit after context cancellation")
	}
}

func Test_flushWorker_multipleBatches(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// Expect two batch inserts: first at batchSize, second with remaining events on timer
	secondDone := make(chan struct{})

	gomock.InOrder(
		mockDB.EXPECT().EventBatchInsert(
			gomock.Any(),
			gomock.Len(batchSize),
		).Return(nil),

		mockDB.EXPECT().EventBatchInsert(
			gomock.Any(),
			gomock.Len(5),
		).DoAndReturn(func(_ interface{}, rows []dbhandler.EventRow) error {
			close(secondDone)
			return nil
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.flushWorker(ctx)

	// Push batchSize + 5 events
	for i := 0; i < batchSize+5; i++ {
		h.eventCh <- &sock.Event{
			Type:      fmt.Sprintf("event_%d", i),
			Publisher: "test-publisher",
			DataType:  "application/json",
			Data:      json.RawMessage(`{}`),
		}
	}

	select {
	case <-secondDone:
		// Both batches flushed
	case <-time.After(3 * time.Second):
		t.Error("flushWorker did not flush the second batch")
	}
}

func Test_flushWorker_drainOnShutdown(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// Push 3 events into the channel before starting the worker
	for i := 0; i < 3; i++ {
		h.eventCh <- &sock.Event{
			Type:      fmt.Sprintf("event_%d", i),
			Publisher: "test-publisher",
			DataType:  "application/json",
			Data:      json.RawMessage(`{}`),
		}
	}

	// Expect exactly one batch insert with 3 events (the final drain flush)
	mockDB.EXPECT().EventBatchInsert(
		gomock.Any(),
		gomock.Len(3),
	).Return(nil)

	// Create an already-cancelled context so flushWorker goes straight to drain
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		h.flushWorker(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Worker drained and exited
	case <-time.After(2 * time.Second):
		t.Error("flushWorker did not drain and exit on cancelled context")
	}
}

func Test_flushWorker_drainOnShutdownEmptyChannel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &subscribeHandler{
		dbHandler: mockDB,
		eventCh:   make(chan *sock.Event, eventChBuffer),
	}

	// No events in the channel — EventBatchInsert should NOT be called.
	// gomock will fail if it gets an unexpected call.

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		h.flushWorker(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Worker exited without flushing
	case <-time.After(2 * time.Second):
		t.Error("flushWorker did not exit on cancelled context with empty channel")
	}
}

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	sh := NewSubscribeHandler(mockSock, mockDB)
	if sh == nil {
		t.Fatal("NewSubscribeHandler returned nil")
	}

	// Verify the internal channel is created with the correct buffer size
	h, ok := sh.(*subscribeHandler)
	if !ok {
		t.Fatal("could not cast to *subscribeHandler")
	}
	if cap(h.eventCh) != eventChBuffer {
		t.Errorf("eventCh capacity = %d, want %d", cap(h.eventCh), eventChBuffer)
	}
}
