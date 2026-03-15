package subscribehandler

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
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
			name: "multiple events",
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
