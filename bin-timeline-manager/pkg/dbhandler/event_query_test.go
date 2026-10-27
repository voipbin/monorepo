package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/event"
)

func Test_EventList(t *testing.T) {
	ts1 := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	ts2 := time.Date(2026, 2, 3, 4, 4, 6, 0, time.UTC)
	resID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	baseSQL := "SELECT timestamp, event_type, publisher, data_type, data FROM events WHERE publisher = ? AND resource_id = ?"
	tests := []struct {
		name string

		events    []string
		pageToken string
		pageSize  int

		queryErr error
		rows     *fakeRows

		wantSQL    string
		wantArgs   []any
		wantErrMsg string
		wantCause  error
		wantClosed int
		wantResult []*event.Event
	}{
		{
			name:       "query error",
			events:     []string{"call_created"},
			pageSize:   5,
			queryErr:   errTestQuery,
			wantSQL:    baseSQL + " AND (event_type = 'call_created') ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), 5},
			wantErrMsg: "could not query events",
			wantCause:  errTestQuery,
		},
		{
			name:     "scan error on the second row",
			pageSize: 5,
			rows: &fakeRows{
				data: [][]any{
					{ts1, "call_created", "call-manager", "call", `{"a":1}`},
					{ts2, "call_hungup", "call-manager", "call", `{"a":2}`},
				},
				scanErr:   errTestScan,
				scanErrAt: 2,
			},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), 5},
			wantErrMsg: "could not scan event row",
			wantCause:  errTestScan,
			wantClosed: 1,
		},
		{
			name:     "iteration error after one row",
			pageSize: 5,
			rows: &fakeRows{
				data: [][]any{{ts1, "call_created", "call-manager", "call", `{"a":1}`}},
				err:  errTestIter,
			},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), 5},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "no rows gives a non-nil empty slice",
			pageSize:   5,
			rows:       &fakeRows{},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), 5},
			wantClosed: 1,
			wantResult: []*event.Event{},
		},
		{
			name:      "wildcard filter adds no condition, page token adds timestamp bound",
			events:    []string{"*"},
			pageToken: "2026-02-03T04:05:06Z",
			pageSize:  2,
			rows: &fakeRows{
				data: [][]any{
					{ts1, "call_created", "call-manager", "call", `{"a":1}`},
					{ts2, "call_hungup", "flow-manager", "flow", `{"b":2}`},
				},
			},
			wantSQL:    baseSQL + " AND timestamp < ? ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), "2026-02-03T04:05:06Z", 2},
			wantClosed: 1,
			wantResult: []*event.Event{
				{Timestamp: ts1, EventType: "call_created", Publisher: "call-manager", DataType: "call", Data: []byte(`{"a":1}`)},
				{Timestamp: ts2, EventType: "call_hungup", Publisher: "flow-manager", DataType: "flow", Data: []byte(`{"b":2}`)},
			},
		},
		{
			name:     "exact and prefix filters",
			events:   []string{"call_created", "flow_*"},
			pageSize: 1,
			rows:     &fakeRows{},
			wantSQL: baseSQL + " AND (event_type = 'call_created' OR event_type LIKE 'flow_%')" +
				" ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"call-manager", resID.String(), 1},
			wantClosed: 1,
			wantResult: []*event.Event{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeConn{queryErr: tt.queryErr}
			if tt.rows != nil {
				tt.rows.t = t
				conn.rows = tt.rows
			}
			h := &dbHandler{conn: conn}

			ctx := context.WithValue(context.Background(), ctxKeyTest{}, "marker")
			got, err := h.EventList(ctx, "call-manager", resID, tt.events, tt.pageToken, tt.pageSize)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
				if got != nil {
					t.Errorf("result = %v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("result is nil, want a non-nil slice")
				}
				if !reflect.DeepEqual(got, tt.wantResult) {
					t.Errorf("result = %+v, want %+v", got, tt.wantResult)
				}
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}
			if normSQL(conn.querySQL) != tt.wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), tt.wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", conn.queryArgs, tt.wantArgs)
			}
			if conn.queryCtx != ctx {
				t.Error("query did not receive the caller's context")
			}
		})
	}
}

func Test_AggregatedEventList(t *testing.T) {
	ts1 := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	ts2 := time.Date(2026, 2, 3, 4, 4, 6, 0, time.UTC)

	baseSQL := "SELECT timestamp, event_type, publisher, data_type, data FROM events WHERE activeflow_id = ?"
	tests := []struct {
		name string

		pageToken string
		pageSize  int

		queryErr error
		rows     *fakeRows

		wantSQL    string
		wantArgs   []any
		wantErrMsg string
		wantCause  error
		wantClosed int
		wantResult []*event.Event
	}{
		{
			name:       "query error",
			pageSize:   9,
			queryErr:   errTestQuery,
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"af-1", 9},
			wantErrMsg: "could not query aggregated events",
			wantCause:  errTestQuery,
		},
		{
			name:     "scan error on the second row",
			pageSize: 9,
			rows: &fakeRows{
				data: [][]any{
					{ts1, "call_created", "call-manager", "call", `{"a":1}`},
					{ts2, "call_hungup", "call-manager", "call", `{"a":2}`},
				},
				scanErr:   errTestScan,
				scanErrAt: 2,
			},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"af-1", 9},
			wantErrMsg: "could not scan event row",
			wantCause:  errTestScan,
			wantClosed: 1,
		},
		{
			name:     "iteration error after one row",
			pageSize: 9,
			rows: &fakeRows{
				data: [][]any{{ts1, "call_created", "call-manager", "call", `{"a":1}`}},
				err:  errTestIter,
			},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"af-1", 9},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "no rows gives a non-nil empty slice",
			pageSize:   9,
			rows:       &fakeRows{},
			wantSQL:    baseSQL + " ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"af-1", 9},
			wantClosed: 1,
			wantResult: []*event.Event{},
		},
		{
			name:      "page token adds timestamp bound and rows are mapped",
			pageToken: "2026-02-03T04:05:06Z",
			pageSize:  2,
			rows: &fakeRows{
				data: [][]any{
					{ts1, "call_created", "call-manager", "call", `{"a":1}`},
					{ts2, "call_hungup", "flow-manager", "flow", `{"b":2}`},
				},
			},
			wantSQL:    baseSQL + " AND timestamp < ? ORDER BY timestamp DESC LIMIT ?",
			wantArgs:   []any{"af-1", "2026-02-03T04:05:06Z", 2},
			wantClosed: 1,
			wantResult: []*event.Event{
				{Timestamp: ts1, EventType: "call_created", Publisher: "call-manager", DataType: "call", Data: []byte(`{"a":1}`)},
				{Timestamp: ts2, EventType: "call_hungup", Publisher: "flow-manager", DataType: "flow", Data: []byte(`{"b":2}`)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeConn{queryErr: tt.queryErr}
			if tt.rows != nil {
				tt.rows.t = t
				conn.rows = tt.rows
			}
			h := &dbHandler{conn: conn}

			ctx := context.WithValue(context.Background(), ctxKeyTest{}, "marker")
			got, err := h.AggregatedEventList(ctx, "af-1", tt.pageToken, tt.pageSize)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
				if got != nil {
					t.Errorf("result = %v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("result is nil, want a non-nil slice")
				}
				if !reflect.DeepEqual(got, tt.wantResult) {
					t.Errorf("result = %+v, want %+v", got, tt.wantResult)
				}
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}
			if normSQL(conn.querySQL) != tt.wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), tt.wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", conn.queryArgs, tt.wantArgs)
			}
			if conn.queryCtx != ctx {
				t.Error("query did not receive the caller's context")
			}
		})
	}
}
