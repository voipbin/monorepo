package dbhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

var (
	errTestQuery = errors.New("injected query failure")
	errTestScan  = errors.New("injected scan failure")
	errTestIter  = errors.New("injected iteration failure")
)

func Test_ResourceActiveflowIDGet(t *testing.T) {
	tests := []struct {
		name string

		queryErr error
		rows     *fakeRows

		wantID      string
		wantErrMsg  string
		wantCause   error
		wantClosed  int
		wantNoError bool
	}{
		{
			name:       "query error",
			queryErr:   errTestQuery,
			wantErrMsg: "could not query resource activeflow_id",
			wantCause:  errTestQuery,
		},
		{
			name:        "no row and no error",
			rows:        &fakeRows{},
			wantNoError: true,
			wantClosed:  1,
		},
		{
			name:       "no row with iteration error",
			rows:       &fakeRows{err: errTestIter},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "scan error",
			rows:       &fakeRows{data: [][]any{{"af-1"}}, scanErr: errTestScan, scanErrAt: 1},
			wantErrMsg: "could not scan activeflow_id",
			wantCause:  errTestScan,
			wantClosed: 1,
		},
		{
			name:        "found",
			rows:        &fakeRows{data: [][]any{{"af-found"}}},
			wantID:      "af-found",
			wantNoError: true,
			wantClosed:  1,
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

			parent, cancel := context.WithCancel(context.Background())
			got, err := h.ResourceActiveflowIDGet(parent, "res-1")

			if tt.wantNoError {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
			}
			if got != tt.wantID {
				t.Errorf("result = %q, want %q", got, tt.wantID)
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}

			if conn.queryCalls != 1 {
				t.Fatalf("Query called %d times, want 1", conn.queryCalls)
			}
			wantSQL := "SELECT activeflow_id FROM events WHERE resource_id = ? AND activeflow_id != '' ORDER BY timestamp ASC LIMIT 1"
			if normSQL(conn.querySQL) != wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, []any{"res-1"}) {
				t.Errorf("args = %v, want [res-1]", conn.queryArgs)
			}

			// The query context is derived from the caller's context, so it
			// carries the settings and follows cancellation.
			if conn.queryCtx == parent {
				t.Error("query context is the parent context, want a derived context")
			}
			cancel()
			if conn.queryCtx.Err() == nil {
				t.Error("query context not cancelled with its parent")
			}
		})
	}
}

func Test_ResourceExists(t *testing.T) {
	tests := []struct {
		name string

		queryErr error
		rows     *fakeRows

		want       bool
		wantErrMsg string
		wantCause  error
		wantClosed int
	}{
		{
			name:       "query error",
			queryErr:   errTestQuery,
			wantErrMsg: "could not query resource existence",
			wantCause:  errTestQuery,
		},
		{
			name:       "no row with iteration error",
			rows:       &fakeRows{err: errTestIter},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "no row",
			rows:       &fakeRows{},
			want:       false,
			wantClosed: 1,
		},
		{
			name:       "row exists",
			rows:       &fakeRows{data: [][]any{{"1"}}},
			want:       true,
			wantClosed: 1,
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

			got, err := h.ResourceExists(context.Background(), "res-2")

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("result = %v, want %v", got, tt.want)
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}

			wantSQL := "SELECT 1 FROM events WHERE resource_id = ? LIMIT 1"
			if normSQL(conn.querySQL) != wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, []any{"res-2"}) {
				t.Errorf("args = %v, want [res-2]", conn.queryArgs)
			}
		})
	}
}

func Test_CorrelatedResourceList(t *testing.T) {
	first1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	last1 := time.Date(2026, 1, 2, 3, 9, 5, 0, time.UTC)
	first2 := time.Date(2026, 1, 3, 4, 5, 6, 0, time.UTC)
	last2 := time.Date(2026, 1, 3, 4, 10, 6, 0, time.UTC)

	tests := []struct {
		name string

		queryErr error
		rows     *fakeRows

		wantErrMsg string
		wantCause  error
		wantLen    int
		wantClosed int
	}{
		{
			name:       "query error",
			queryErr:   errTestQuery,
			wantErrMsg: "could not query correlated resources",
			wantCause:  errTestQuery,
		},
		{
			name: "scan error on the second row",
			rows: &fakeRows{
				data: [][]any{
					{"call-manager", "res-1", "call", []string{"a"}, first1, last1},
					{"flow-manager", "res-2", "flow", []string{"b"}, first2, last2},
				},
				scanErr:   errTestScan,
				scanErrAt: 2,
			},
			wantErrMsg: "could not scan correlated row",
			wantCause:  errTestScan,
			wantClosed: 1,
		},
		{
			name: "iteration error after one row",
			rows: &fakeRows{
				data: [][]any{{"call-manager", "res-1", "call", []string{"a"}, first1, last1}},
				err:  errTestIter,
			},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "no rows",
			rows:       &fakeRows{},
			wantLen:    0,
			wantClosed: 1,
		},
		{
			name: "two rows",
			rows: &fakeRows{
				data: [][]any{
					{"call-manager", "res-1", "call", []string{"call_created", "call_hungup"}, first1, last1},
					{"flow-manager", "res-2", "flow", []string{"flow_created"}, first2, last2},
				},
			},
			wantLen:    2,
			wantClosed: 1,
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

			parent, cancel := context.WithCancel(context.Background())
			got, err := h.CorrelatedResourceList(parent, "af-1", 7)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
				if got != nil {
					t.Errorf("result = %v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil || len(got) != tt.wantLen {
					t.Errorf("result = %v, want non-nil slice of %d", got, tt.wantLen)
				}
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}

			wantSQL := "SELECT publisher, resource_id, min(data_type) AS data_type, groupUniqArray(event_type) AS event_types, " +
				"min(timestamp) AS first_seen, max(timestamp) AS last_seen FROM events WHERE activeflow_id = ? AND resource_id != '' " +
				"GROUP BY publisher, resource_id ORDER BY first_seen ASC LIMIT ?"
			if normSQL(conn.querySQL) != wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, []any{"af-1", 7}) {
				t.Errorf("args = %v, want [af-1 7]", conn.queryArgs)
			}

			if conn.queryCtx == parent {
				t.Error("query context is the parent context, want a derived context")
			}
			cancel()
			if conn.queryCtx.Err() == nil {
				t.Error("query context not cancelled with its parent")
			}
		})
	}
}

func Test_CorrelatedResourceList_FieldMapping(t *testing.T) {
	first1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	last1 := time.Date(2026, 1, 2, 3, 9, 5, 0, time.UTC)
	first2 := time.Date(2026, 1, 3, 4, 5, 6, 0, time.UTC)
	last2 := time.Date(2026, 1, 3, 4, 10, 6, 0, time.UTC)

	conn := &fakeConn{rows: &fakeRows{
		t: t,
		data: [][]any{
			{"call-manager", "res-1", "call", []string{"call_created", "call_hungup"}, first1, last1},
			{"flow-manager", "res-2", "flow", []string{"flow_created"}, first2, last2},
		},
	}}
	h := &dbHandler{conn: conn}

	got, err := h.CorrelatedResourceList(context.Background(), "af-1", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	if g := got[0]; g.Publisher != "call-manager" || g.ResourceID != "res-1" || g.DataType != "call" ||
		!reflect.DeepEqual(g.EventTypes, []string{"call_created", "call_hungup"}) ||
		!g.FirstSeen.Equal(first1) || !g.LastSeen.Equal(last1) {
		t.Errorf("row 0 = %+v", g)
	}
	if g := got[1]; g.Publisher != "flow-manager" || g.ResourceID != "res-2" || g.DataType != "flow" ||
		!reflect.DeepEqual(g.EventTypes, []string{"flow_created"}) ||
		!g.FirstSeen.Equal(first2) || !g.LastSeen.Equal(last2) {
		t.Errorf("row 1 = %+v", g)
	}
}
